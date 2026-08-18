import {Dialog, DialogClose, DialogContent, DialogFooter, DialogTrigger} from "~/components/ui/dialog";
import {Button} from "~/components/ui/button";

function PopupDocsRA({docs, name}: {docs: string, name: string}) {
//   popup docs for read and access
//   TODO: Сделать общий для всех новый класс docs с названиями всех документов чтобы не путаться
//   TODO: М.б. почитать как сделать запрос сразу при заходе чтобы обрабатывался он дабы пользователь не ждал
  return (
    <Dialog>
      <form action="">
        <DialogTrigger render={<Button variant="link" className="p-1">{name}</Button>} />
        {/*Функция для обработки json в документ*/}
        <DialogContent>
          <DialogFooter>
            <DialogClose render={<Button className="w-full">Закрыть</Button>} />
          </DialogFooter>
        </DialogContent>
      </form>
    </Dialog>
  )
}
export default PopupDocsRA;
//JSON должен иметь вид
// Title
// Description
//JSONB:
// А тут придумать потом как делать подтайтл, подподтайтл, список, сделать доки и посмотреть потом как разбивать

//Так примерно будет:
// - popup обобщенный с кнопкой (согласится)
// - все виды документов с которыми надо согласится или парсер для json документа в теле карты. Принимать должен получается название документа обобщенное и выкатывать последнюю версию (go вышлет последнею версию)