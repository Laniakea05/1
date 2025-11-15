CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица психологических тестов
CREATE TABLE IF NOT EXISTS psychological_tests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    instructions TEXT,
    estimated_time INTEGER,
    is_active BOOLEAN DEFAULT true,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица вопросов теста
CREATE TABLE IF NOT EXISTS test_questions (
    id SERIAL PRIMARY KEY,
    test_id INTEGER REFERENCES psychological_tests(id),
    question_text TEXT NOT NULL,
    question_type VARCHAR(50) DEFAULT 'multiple_choice',
    options JSONB,
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
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    answers JSONB,
    interpretation TEXT
);

-- Создаём индексы для производительности
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_test_results_user_id ON test_results(user_id);
CREATE INDEX IF NOT EXISTS idx_test_results_test_id ON test_results(test_id);
CREATE INDEX IF NOT EXISTS idx_test_questions_test_id ON test_questions(test_id);

-- Пример теста (без привязки к пользователю, так как пользователей ещё нет)
INSERT INTO psychological_tests (title, description, instructions, estimated_time) 
VALUES (
    'Тест на стрессоустойчивость', 
    'Оценка способности работать в стрессовых ситуациях, что критично для специалистов по информационной безопасности', 
    'Внимательно прочитайте каждый вопрос и выберите наиболее подходящий вариант ответа. Отвечайте честно, не задумываясь слишком долго.', 
    15
) ON CONFLICT DO NOTHING;

-- Вопросы для теста
INSERT INTO test_questions (test_id, question_text, question_type, options, order_index) 
VALUES 
(1, 'Как вы реагируете, когда обнаруживаете попытку взлома системы?', 'multiple_choice', '["Паникую", "Спокойно анализирую ситуацию", "Немедленно сообщаю руководству", "Пытаюсь скрыть инцидент"]', 1),
(1, 'При высокой рабочей нагрузке вы:', 'multiple_choice', '["Эффективно планируете задачи", "Работаете в состоянии стресса", "Откладываете сложные задачи", "Просите помощи у коллег"]', 2),
(1, 'Ваша реакция на критику:', 'multiple_choice', '["Принимаю к сведению", "Защищаюсь и оправдываюсь", "Игнорирую", "Анализирую и делаю выводы"]', 3)
ON CONFLICT DO NOTHING;