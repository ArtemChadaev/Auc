BEGIN;

-- Вставка нормативных документов платформы Auc из файлов на диске
-- Чтение содержимого каждого файла осуществляется непосредственно через функцию pg_read_file
WITH doc_sources AS (
    SELECT
        name,
        version,
        is_major,
        COALESCE(
            pg_read_file('documents/' || file_ru, 0, 10000000, true),
            pg_read_file('/migrations/documents/' || file_ru, 0, 10000000, true),
            pg_read_file('migrations/documents/' || file_ru, 0, 10000000, true),
            pg_read_file('C:/Users/Chadaev/GolandProjects/Auc/migrations/documents/' || file_ru, 0, 10000000, true),
            pg_read_file(file_ru, 0, 10000000, true),
            pg_read_file('documents/' || file_en, 0, 10000000, true),
            pg_read_file('/migrations/documents/' || file_en, 0, 10000000, true),
            pg_read_file('migrations/documents/' || file_en, 0, 10000000, true),
            pg_read_file('C:/Users/Chadaev/GolandProjects/Auc/migrations/documents/' || file_en, 0, 10000000, true),
            pg_read_file(file_en, 0, 10000000, true)
        ) AS doc_text
    FROM (
        VALUES
            ('terms-of-service', '1.0', true, 'terms-of-service.ru.md', 'terms-of-service.md'),
            ('privacy-policy', '1.0', true, 'privacy-policy.ru.md', 'privacy-policy.md'),
            ('virtual-currency-disclaimer', '1.0', true, 'virtual-currency-disclaimer.ru.md', 'virtual-currency-disclaimer.md'),
            ('auction-rules', '1.0', true, 'auction-rules.ru.md', 'auction-rules.md'),
            ('content-policy', '1.0', true, 'content-policy.ru.md', 'content-policy.md'),
            ('acceptable-use-policy', '1.0', true, 'acceptable-use-policy.ru.md', 'acceptable-use-policy.md'),
            ('cookie-policy', '1.0', false, 'cookie-policy.ru.md', 'cookie-policy.md')
    ) AS t(name, version, is_major, file_ru, file_en)
)
INSERT INTO documents (name, version, is_major, text)
SELECT
    name,
    version,
    is_major,
    doc_text
FROM doc_sources
ON CONFLICT (name, version) DO UPDATE
SET is_major = EXCLUDED.is_major,
    text = EXCLUDED.text,
    superseded_at = NULL;

COMMIT;
