Configured authentication: Managed by agent

Анализ представленного кода выявил ряд критических проблем: от ошибок компиляции и уязвимостей к отказу в обслуживании (DoS / OOM) до гонок при проверке размеров и искажения геометрии изображений.

---

### 1. Ошибки компиляции и несоответствия типов

#### `getModel3DMetadata`: сравнение `uint32` и `int`
В библиотеке `github.com/qmuntal/gltf`:
* `primitive.Attributes[gltf.POSITION]` возвращает `uint32`.
* `*primitive.Indices` имеет тип `uint32`.
* Функция `len(doc.Accessors)` возвращает встроенный `int`.

В Go прямое сравнение разных типов без явного приведения запрещено:
```go
// ОШИБКА КОМПИЛЯЦИИ: invalid operation: posIdx < len(doc.Accessors) (mismatched types uint32 and int)
if hasPos && posIdx < len(doc.Accessors) { ... }

// ОШИБКА КОМПИЛЯЦИИ: invalid operation: idx < len(doc.Accessors)
if idx < len(doc.Accessors) { ... }
```
**Решение:** использовать явное приведение `int(posIdx) < len(doc.Accessors)` или `posIdx < uint32(len(doc.Accessors))`.

---

### 2. Безопасность и DoS / OOM (Out Of Memory)

#### `getDocumentMetadata`: чтение файла целиком без лимита
```go
data, err := io.ReadAll(r)
```
* Если пользователь загружает PDF или DOCX размером 1–2 ГБ, `io.ReadAll` загрузит весь файл в оперативную память. Несколько одновременных запросов приведут к падению сервиса по OOM.
* **Решение:** использовать `io.LimitReader(r, maxDocumentBytes)` или временный файл на диске (`os.CreateTemp`), если требуется случайный доступ (`io.ReaderAt` / `io.ReadSeeker`).

#### `getArchiveMetadata`: неконтролируемое накопление слайса `Tree` (Zip-бомба)
```go
tree = append(tree, name)
```
* В архиве может находиться 500 000+ файлов (или это может быть архив-бомба).
* Все имена складываются в неограниченный слайс `tree []string`, который затем маршалится в JSON и пишется в PostgreSQL. Это вызовет гигантский расход RAM и сбой при вставке в БД.
* **Решение:** ограничить максимальное количество файлов в `Tree` (например, первые 100–500 файлов), лимитировать `fileCount` и суммарный `uncompressedSizeBytes`. Если лимит превышен — возвращать ошибку `apperr.ErrEntityTooLarge`.

#### `getImageMetadata`: гонка при защите от декомпрессионных бомб (OOM)
Горутина с `image.Decode(pr1)` и горутина с `image.DecodeConfig(pr2)` запущены **одновременно**:
```go
g.Go(func() { srcImg, _, err := image.Decode(pr1) ... })
g.Go(func() { config, _, err := image.DecodeConfig(pr2) ... })
```
* `io.Copy(MultiWriter(pw1, pw2), r)` синхронно льет поток в обе трубы.
* Пока `DecodeConfig` читает заголовок и валидирует размеры, `image.Decode` уже начинает декодировать и аллоцировать память под пиксели. При загрузке изображения с огромными габаритами (пиксельная бомба) `image.Decode` успеет вызвать OOM еще до того, как вторая горутина вернет ошибку.
* **Переполнение `int`:** проверка `config.Width * config.Height * 4 > 50<<20` на 32-битных системах или при значениях вроде $65536 \times 65536$ переполнит знаковый `int` в 0 и обойдет ограничение.
* **Решение:** проверять размер **до** декодирования. Например, через `bufio.Reader` (подглядеть заголовок `image.DecodeConfig`, проверить габариты, затем декодировать без горутин и пайпов).

---

### 3. Алгоритмические ошибки и искажение пропорций

#### `getImageMetadata`: целочисленное деление и нулевая ширина/высота
```go
if height > 100 || width > 100 {
    if width >= height {
        height = height / (width / 100)
        width = 100
    } else {
        width = width / (height / 100)
        height = 100
    }
}
```
1. **Искажение пропорций:** деление `width / 100` — целочисленное!
    * Если `width = 199`, `height = 100`: `199 / 100 == 1`, поэтому `height = 100 / 1 = 100`, а `width = 100`. Исходное изображение было вытянутым по ширине (199x100), а стало квадратным (100x100).
    * Если `width = 250`, `height = 200`: `250 / 100 == 2`, `height = 200 / 2 = 100`, `width = 100`. Пропорция 1.25 превратилась в 1.0.
2. **Нулевой размер и паника `blurhash`:**
    * Если `width = 1`, `height = 200`: ветка `else` выполняет `height / 100 == 2`, а затем `width = 1 / 2 = 0`.
    * Создается прямоугольник `image.Rect(0, 0, 0, 100)`. `blurhash.Encode` при нулевой ширине упадет с паникой или вернет ошибку.
3. **Решение:** использовать коэффициент масштабирования через `float64`:
   ```go
   scale := math.Min(100.0/float64(width), 100.0/float64(height))
   newW := max(1, int(math.Round(float64(width)*scale)))
   newH := max(1, int(math.Round(float64(height)*scale)))
   ```

---

### 4. Проблемы I/O, конкурентности и контекста

#### `getImageMetadata`: избыточность пайпов и тип `*io.PipeReader`
* Функция жестко завязана на `r *io.PipeReader` вместо `r io.Reader`. Это ломает абстракцию и делает невозможным модульное тестирование с `bytes.Reader` или `os.File`.
* Создание 2 пайпов, 3 горутин и `MultiWriter` для чтения одного изображения создает риск блокировок: если один из читателей замедлится, `MultiWriter` заблокирует запись второму.
* **Ветка SVG:**
  ```go
  if mType == "image/svg+xml" {
      return metadata, nil
  }
  ```
  Функция завершается, вообще не вычитывая `r`. В родительской функции `itemService.go` `prMData` спасен только вызовом `io.Copy(io.Discard, prMData)` после switch, но для изолированной функции такое поведение некорректно.

#### `getModel3DMetadata`: игнорирование `ctx`
* `ctx context.Context` передан первым аргументом, но внутри нигде не используется. Если поток `r` зависнет, отмена родительского контекста не прервет выполнение `decoder.Decode(doc)`.

---

### 5. Специфика форматов и внешние зависимости

#### `getAudioMetadata` и `getVideoMetadata`: зависимость от `ffprobe` и стриминг MP4
1. **Внешний бинарник:** `ffprobe.ProbeReader` запускает системную утилиту через `os/exec`. Если `ffprobe` не установлен в системе или Docker-контейнере, все вызовы завершатся ошибкой `executable file not found in $PATH`.
2. **Невозможность Seek в потоке:** `ProbeReader` передает `r` через `stdin`. В большинстве MP4/MOV файлов метаданные (`moov` atom) по умолчанию записаны в **конце** файла (если не применялся `qt-faststart`). Потоковый `stdin` нельзя перемотать назад, поэтому `ffprobe` либо будет вычитывать гигабайтный файл до конца, либо завершится с ошибкой `moov atom not found`.

#### `getOfficePageCount`: ненадежность метаданных DOCX/PPTX
1. **Регистр имени:** `f.Name == "docProps/app.xml"` чувствителен к регистру. В некоторых архиваторах путь может быть `DocProps/app.xml` или `docprops/app.xml`. Надежнее проверять через `strings.EqualFold`.
2. **Необязательность тега `<Pages>`:** Сторонние офисные редакторы (LibreOffice, Google Docs, Apple Pages) часто не заполняют или не обновляют теги `<Pages>` в `docProps/app.xml`. Возврат `1` в таких случаях может дезинформировать пользователя.

#### `getModel3DMetadata`: режим примитива по умолчанию
* В спецификации glTF 2.0 поле `mode` у `primitive` является опциональным и по умолчанию равно `4` (`PrimitiveTriangles`). Если парсер оставляет его дефолтным нулем Go (`PrimitivePoints`), условие `switch primitive.Mode` не сработает ни для одного `case`, и `polygonCount` останется равным `0`.

---

### 6. Критический баг в вызывающем коде (`itemService.go`)

В файле [itemService.go](file:///C:/Users/Chadaev/GolandProjects/Auc/internal/item/itemService.go#L83-L90) найдена опечатка в горутине вычисления хеша:
```go
g.Go(func() (err error) {
    defer func() {
        _ = prMData.CloseWithError(err) // ОШИБКА: закрывается prMData вместо prHash!
    }()

    if _, err = io.Copy(hasher, prHash); err != nil { ... }
    return
})
```
Горутина хешера читает из `prHash`, но в `defer` закрывает `prMData`! Если горутина хеширования завершится с ошибкой, она оборвет пайп метаданных, а `prHash` останется незакрытым.