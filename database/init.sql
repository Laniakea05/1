DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS test_questions;
DROP TABLE IF EXISTS psychological_tests;
DROP TABLE IF EXISTS users;

-- Таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    is_blocked BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица психологических тестов
CREATE TABLE psychological_tests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    instructions TEXT,
    estimated_time INTEGER,
    is_active BOOLEAN DEFAULT true,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица вопросов теста (обновленная структура)
CREATE TABLE test_questions (
    id SERIAL PRIMARY KEY,
    test_id INTEGER REFERENCES psychological_tests(id),
    question_text TEXT NOT NULL,
    question_type VARCHAR(50) DEFAULT 'multiple_choice',
    options JSONB NOT NULL,
    weight DECIMAL(3,2) DEFAULT 1.0,
    order_index INTEGER
);

-- Таблица результатов тестирования (обновленная)
CREATE TABLE test_results (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    test_id INTEGER REFERENCES psychological_tests(id),
    score DECIMAL(5,2),
    max_score DECIMAL(5,2),
    interpretation TEXT,
    recommendation TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    answers JSONB
);

-- Создаём индексы для производительности
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_test_results_user_id ON test_results(user_id);
CREATE INDEX idx_test_results_test_id ON test_results(test_id);
CREATE INDEX idx_test_questions_test_id ON test_questions(test_id);
CREATE INDEX idx_test_results_completed_at ON test_results(completed_at);
CREATE INDEX idx_users_is_blocked ON users(is_blocked);

-- УБИРАЕМ ВСТАВКУ ПОЛЬЗОВАТЕЛЕЙ ОТСЮДА! Они создадутся через функцию CreateTestUsers()

-- Пример теста с весами ответов от 1 до 5
INSERT INTO psychological_tests (title, description, instructions, estimated_time) 
VALUES (
    'Тест на стрессоустойчивость', 
    'Оценка способности работать в стрессовых ситуациях', 
    'Внимательно прочитайте каждый вопрос и выберите наиболее подходящий вариант ответа. Отвечайте честно, не задумываясь слишком долго. Результаты помогут оценить вашу психологическую устойчивость в рабочих ситуациях.', 
    15
) ON CONFLICT DO NOTHING;

-- Вопросы для теста с весами ответов от 1 до 5
INSERT INTO test_questions (test_id, question_text, question_type, options, order_index) 
VALUES 
(1, 'Как вы реагируете, когда обнаруживаете попытку взлома системы?', 'multiple_choice', 
 '[{"text": "Паникую и теряю контроль над ситуацией", "weight": 1}, 
   {"text": "Немного нервничаю, но стараюсь действовать", "weight": 2}, 
   {"text": "Немедленно сообщаю руководству и жду указаний", "weight": 3}, 
   {"text": "Соблюдаю установленные процедуры реагирования", "weight": 4}, 
   {"text": "Спокойно анализирую ситуацию и действую по инструкции", "weight": 5}]', 1),

(1, 'При высокой рабочей нагрузке вы:', 'multiple_choice', 
 '[{"text": "Теряю концентрацию и допускаю ошибки", "weight": 1}, 
   {"text": "Работаю в состоянии стресса, что снижает эффективность", "weight": 2}, 
   {"text": "Откладываю сложные задачи на потом", "weight": 3}, 
   {"text": "Просите помощи у коллег или руководства", "weight": 4}, 
   {"text": "Эффективно планируете задачи и распределяете время", "weight": 5}]', 2),

(1, 'Ваша реакция на критику работы:', 'multiple_choice', 
 '[{"text": "Расстраиваюсь и теряю мотивацию", "weight": 1}, 
   {"text": "Защищаюсь и оправдываю свои действия", "weight": 2}, 
   {"text": "Выслушиваю, но не всегда принимаю к сведению", "weight": 3}, 
   {"text": "Анализирую критику и делаю выводы для улучшения", "weight": 4}, 
   {"text": "Принимаю к сведению и активно работаю над ошибками", "weight": 5}]', 3),

(1, 'Как вы справляетесь с нештатными ситуациями?', 'multiple_choice', 
 '[{"text": "Теряюсь и не знаю что делать", "weight": 1}, 
   {"text": "Действую хаотично, без четкого плана", "weight": 2}, 
   {"text": "Обращаюсь за помощью к коллегам", "weight": 3}, 
   {"text": "Следую инструкциям, но с некоторым напряжением", "weight": 4}, 
   {"text": "Сохраняю спокойствие и действую по отработанным процедурам", "weight": 5}]', 4),

(1, 'Ваше отношение к срокам выполнения задач:', 'multiple_choice', 
 '[{"text": "Часто не успеваю, что вызывает стресс", "weight": 1}, 
   {"text": "Работаю в последний момент, с большим напряжением", "weight": 2}, 
   {"text": "Стараюсь уложиться, но иногда переношу сроки", "weight": 3}, 
   {"text": "Планирую время и обычно укладываюсь в сроки", "weight": 4}, 
   {"text": "Всегда завершаю задачи заранее, с запасом времени", "weight": 5}]', 5),

(1, 'Как вы восстанавливаетесь после рабочего дня?', 'multiple_choice', 
 '[{"text": "Не могу отключиться от рабочих мыслей", "weight": 1}, 
   {"text": "Нужно несколько часов чтобы прийти в себя", "weight": 2}, 
   {"text": "Отдыхаю, но иногда думаю о работе", "weight": 3}, 
   {"text": "Умею переключаться на личные дела", "weight": 4}, 
   {"text": "Эффективно отделяю работу от личной жизни", "weight": 5}]', 6),

(1, 'Ваша реакция на конфликтные ситуации в коллективе:', 'multiple_choice', 
 '[{"text": "Избегаю конфликтов любой ценой", "weight": 1}, 
   {"text": "Сильно переживаю, теряю работоспособность", "weight": 2}, 
   {"text": "Стараюсь не участвовать, сохраняя нейтралитет", "weight": 3}, 
   {"text": "Пытаюсь найти компромиссное решение", "weight": 4}, 
   {"text": "Конструктивно подхожу к разрешению конфликта", "weight": 5}]', 7),

(1, 'Как вы относитесь к изменениям в рабочих процессах?', 'multiple_choice', 
 '[{"text": "Изменения вызывают сильный стресс", "weight": 1}, 
   {"text": "С трудом адаптируюсь к новому", "weight": 2}, 
   {"text": "Принимаю изменения после периода адаптации", "weight": 3}, 
   {"text": "Отношусь к изменениям как к возможности развития", "weight": 4}, 
   {"text": "Легко адаптируюсь и активно участвую в изменениях", "weight": 5}]', 8),

(1, 'Ваше эмоциональное состояние в конце рабочей недели:', 'multiple_choice', 
 '[{"text": "Полное эмоциональное истощение", "weight": 1}, 
   {"text": "Сильная усталость, нужен длительный отдых", "weight": 2}, 
   {"text": "Усталость, но могу восстановиться за выходные", "weight": 3}, 
   {"text": "Легкая усталость, хорошее настроение сохраняется", "weight": 4}, 
   {"text": "Бодрое состояние, готов к новым задачам", "weight": 5}]', 9),

(1, 'Как вы оцениваете свою способность к многозадачности?', 'multiple_choice', 
 '[{"text": "Не справляюсь с несколькими задачами одновременно", "weight": 1}, 
   {"text": "Многозадачность вызывает сильный стресс", "weight": 2}, 
   {"text": "Справляюсь, но с некоторым напряжением", "weight": 3}, 
   {"text": "Умею распределять внимание между задачами", "weight": 4}, 
   {"text": "Легко переключаюсь между задачами без потери эффективности", "weight": 5}]', 10)
ON CONFLICT DO NOTHING;

-- Второй тест для демонстрации
INSERT INTO psychological_tests (title, description, instructions, estimated_time) 
VALUES (
    'Тест на эмоциональное выгорание', 
    'Диагностика уровня профессионального выгорания', 
    'Оцените, насколько часто вы испытываете перечисленные состояния и чувства в вашей профессиональной деятельности. Выбирайте ответы, которые наиболее точно отражают ваши переживания.', 
    10
) ON CONFLICT DO NOTHING;

-- Вопросы для второго теста
INSERT INTO test_questions (test_id, question_text, question_type, options, order_index) 
VALUES 
(2, 'Как часто вы чувствуете усталость в начале рабочего дня?', 'multiple_choice', 
 '[{"text": "Постоянно, каждый день", "weight": 1}, 
   {"text": "Очень редко", "weight": 2}, 
   {"text": "Иногда", "weight": 3}, 
   {"text": "Часто", "weight": 4}, 
   {"text": "Ежедневно", "weight": 5}]', 1),

(2, 'Мне кажется, что я стал/стала более черствым/ой по отношению к людям на работе', 'multiple_choice', 
 '[{"text": "Никогда", "weight": 1}, 
   {"text": "Очень редко", "weight": 2}, 
   {"text": "Иногда", "weight": 3}, 
   {"text": "Часто", "weight": 4}, 
   {"text": "Ежедневно", "weight": 5}]', 2),

(2, 'Я сомневаюсь в значимости и важности своей работы', 'multiple_choice', 
 '[{"text": "Никогда", "weight": 1}, 
   {"text": "Очень редко", "weight": 2}, 
   {"text": "Иногда", "weight": 3}, 
   {"text": "Часто", "weight": 4}, 
   {"text": "Постоянно", "weight": 5}]', 3)
ON CONFLICT DO NOTHING;

-- УБИРАЕМ тестовые результаты, так как пользователей еще нет
-- Они создадутся после запуска системы

-- Комментарии к таблицам для документации
COMMENT ON TABLE users IS 'Таблица пользователей системы';
COMMENT ON TABLE psychological_tests IS 'Таблица психологических тестов';
COMMENT ON TABLE test_questions IS 'Таблица вопросов тестов с вариантами ответов и весами';
COMMENT ON TABLE test_results IS 'Таблица результатов прохождения тестов';

COMMENT ON COLUMN users.is_blocked IS 'Флаг блокировки пользователя';
COMMENT ON COLUMN test_questions.options IS 'JSON массив объектов с текстом ответа и весом (1-5)';
COMMENT ON COLUMN test_results.interpretation IS 'Текстовая интерпретация результата теста';
COMMENT ON COLUMN test_results.recommendation IS 'Рекомендации по результатам тестирования';