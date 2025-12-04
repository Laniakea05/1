DROP TABLE IF EXISTS user_answers;
DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS test_questions;
DROP TABLE IF EXISTS psychological_tests;
DROP TABLE IF EXISTS users;

-- Таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash CHAR(60) NOT NULL,
    last_name VARCHAR(30) NOT NULL,
    first_name VARCHAR(30) NOT NULL,
    patronymic VARCHAR(30),
    role VARCHAR(10) DEFAULT 'user',
    is_blocked BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица психологических тестов для ИБ специалистов
CREATE TABLE psychological_tests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    instructions TEXT,
    estimated_time INTEGER,
    is_active BOOLEAN DEFAULT true,
    pass_threshold DECIMAL(5,2) NOT NULL DEFAULT 70.0,
    methodology_type VARCHAR(30) NOT NULL,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица вопросов теста
CREATE TABLE test_questions (
    id SERIAL PRIMARY KEY,
    test_id INTEGER REFERENCES psychological_tests(id),
    question_text TEXT NOT NULL,
    question_type VARCHAR(50) DEFAULT 'multiple_choice',
    scale_type VARCHAR(100) NOT NULL,
    weight DECIMAL(3,2) DEFAULT 1.0,
    order_index INTEGER
);

-- Таблица вариантов ответов
CREATE TABLE question_options (
    id SERIAL PRIMARY KEY,
    question_id INTEGER REFERENCES test_questions(id),
    option_text TEXT NOT NULL,
    score_value INTEGER NOT NULL DEFAULT 0,
    order_index INTEGER
);

-- Таблица результатов тестирования
CREATE TABLE test_results (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    test_id INTEGER REFERENCES psychological_tests(id),
    total_score DECIMAL(5,2) NOT NULL,
    max_possible_score DECIMAL(5,2) NOT NULL,
    percentage DECIMAL(5,2) NOT NULL,
    is_passed BOOLEAN NOT NULL,
    interpretation TEXT NOT NULL,
    recommendation TEXT,
    scale_results JSONB,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица ответов пользователя
CREATE TABLE user_answers (
    id SERIAL PRIMARY KEY,
    result_id INTEGER REFERENCES test_results(id),
    question_id INTEGER REFERENCES test_questions(id),
    option_id INTEGER REFERENCES question_options(id),
    answered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создаём индексы для производительности
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_test_results_user_id ON test_results(user_id);
CREATE INDEX idx_test_results_test_id ON test_results(test_id);
CREATE INDEX idx_test_questions_test_id ON test_questions(test_id);
CREATE INDEX idx_question_options_question_id ON question_options(question_id);
CREATE INDEX idx_user_answers_result_id ON user_answers(result_id);
CREATE INDEX idx_test_results_completed_at ON test_results(completed_at);
CREATE INDEX idx_users_is_blocked ON users(is_blocked);

-- Обновляем ограничения внешних ключей для поддержки SET NULL
ALTER TABLE user_answers 
DROP CONSTRAINT IF EXISTS user_answers_option_id_fkey,
DROP CONSTRAINT IF EXISTS user_answers_question_id_fkey,
ADD CONSTRAINT user_answers_option_id_fkey 
FOREIGN KEY (option_id) 
REFERENCES question_options(id) 
ON DELETE SET NULL,
ADD CONSTRAINT user_answers_question_id_fkey 
FOREIGN KEY (question_id) 
REFERENCES test_questions(id) 
ON DELETE SET NULL;

ALTER TABLE test_results 
DROP CONSTRAINT IF EXISTS test_results_test_id_fkey,
ADD CONSTRAINT test_results_test_id_fkey 
FOREIGN KEY (test_id) 
REFERENCES psychological_tests(id) 
ON DELETE SET NULL;

-- Тест 1: Методика измерения ригидности (модифицированная для ИБ)
INSERT INTO psychological_tests (title, description, instructions, estimated_time, pass_threshold, methodology_type) VALUES 
('Методика измерения ригидности',
'Оценка психологической гибкости, способности адаптироваться к изменениям и переключаться между задачами',
'Отвечайте "Да" или "Нет" на следующие утверждения. Выбирайте тот вариант, который наиболее точно отражает ваше обычное поведение.',
15, 60.0, 'rigidity_scale');

-- Вопросы для методики ригидности (все с прямой шкалой)
INSERT INTO test_questions (test_id, question_text, scale_type, order_index) VALUES 
(1, 'Мне трудно менять свои привычки даже когда этого требуют обстоятельства', 'cognitive_rigidity', 1),
(1, 'Я легко переключаюсь с одной задачи на другую', 'behavioral_flexibility', 2),
(1, 'Новые методы работы вызывают у меня сопротивление и раздражение', 'innovation_resistance', 3),
(1, 'Я предпочитаю работать по проверенным схемам, а не экспериментировать', 'procedural_preference', 4),
(1, 'Мне сложно адаптироваться к изменениям в рабочих процессах', 'adaptability', 5),
(1, 'Я быстро осваиваю новые технологии и инструменты', 'learning_ability', 6),
(1, 'Мне нравится, когда все идет по плану без неожиданностей', 'planning_preference', 7),
(1, 'Я легко нахожу альтернативные решения при возникновении проблем', 'problem_solving', 8),
(1, 'Изменения в распорядке дня выводят меня из равновесия', 'emotional_stability', 9),
(1, 'Я готов изменить свое мнение под влиянием новых фактов', 'openness_to_change', 10);

-- Варианты ответов для методики ригидности (прямая шкала)
-- Высокие баллы = низкая ригидность (хорошо для ИБ)
INSERT INTO question_options (question_id, option_text, score_value, order_index) VALUES 
-- Вопрос 1: Мне трудно менять свои привычки (НЕТ = 2 балла, ДА = 0 баллов)
(1, 'Да', 0, 1),
(1, 'Нет', 2, 2),

-- Вопрос 2: Я легко переключаюсь с одной задачи на другую (ДА = 2 балла, НЕТ = 0 баллов)
(2, 'Да', 2, 1),
(2, 'Нет', 0, 2),

-- Вопрос 3: Новые методы вызывают сопротивление (НЕТ = 2 балла, ДА = 0 баллов)
(3, 'Да', 0, 1),
(3, 'Нет', 2, 2),

-- Вопрос 4: Предпочитаю проверенные схемы (НЕТ = 2 балла, ДА = 0 баллов)
(4, 'Да', 0, 1),
(4, 'Нет', 2, 2),

-- Вопрос 5: Сложно адаптироваться к изменениям (НЕТ = 2 балла, ДА = 0 баллов)
(5, 'Да', 0, 1),
(5, 'Нет', 2, 2),

-- Вопрос 6: Быстро осваиваю новые технологии (ДА = 2 балла, НЕТ = 0 баллов)
(6, 'Да', 2, 1),
(6, 'Нет', 0, 2),

-- Вопрос 7: Люблю когда все по плану (НЕТ = 2 балла, ДА = 0 баллов)
(7, 'Да', 0, 1),
(7, 'Нет', 2, 2),

-- Вопрос 8: Легко нахожу альтернативные решения (ДА = 2 балла, НЕТ = 0 баллов)
(8, 'Да', 2, 1),
(8, 'Нет', 0, 2),

-- Вопрос 9: Изменения выводят из равновесия (НЕТ = 2 балла, ДА = 0 баллов)
(9, 'Да', 0, 1),
(9, 'Нет', 2, 2),

-- Вопрос 10: Готов изменить мнение (ДА = 2 балла, НЕТ = 0 баллов)
(10, 'Да', 2, 1),
(10, 'Нет', 0, 2);

-- Тест 2: Опросник волевого самоконтроля (ВСК) - все вопросы прямые
INSERT INTO psychological_tests (title, description, instructions, estimated_time, pass_threshold, methodology_type) VALUES 
('Опросник волевого самоконтроля (ВСК)',
'Диагностика уровня развития волевой регуляции и самоконтроля в профессиональной деятельности',
'Оцените, насколько приведенные утверждения соответствуют вашему поведению. Выбирайте ответы по шкале от "Полностью не согласен" до "Полностью согласен".',
20, 70.0, 'willpower_control');

-- Вопросы для опросника ВСК (все с прямой шкалой, все позитивно сформулированы)
INSERT INTO test_questions (test_id, question_text, scale_type, order_index) VALUES 
(2, 'Я всегда довожу начатое дело до конца', 'perseverance', 1),
(2, 'Мне легко заставить себя работать через "не хочу"', 'self_motivation', 2),
(2, 'Я приступаю к сложным задачам сразу, не откладывая', 'procrastination', 3),
(2, 'В критических ситуациях я сохраняю спокойствие и самообладание', 'emotional_control', 4),
(2, 'Я могу долго работать, не отвлекаясь на посторонние вещи', 'concentration', 5),
(2, 'Мне легко соблюдать установленный распорядок дня', 'discipline', 6),
(2, 'Я всегда выполняю данные обещания', 'responsibility', 7),
(2, 'При неудачах я продолжаю работать над задачей', 'resilience', 8),
(2, 'Я могу заставить себя делать то, что необходимо, даже если это неприятно', 'self_discipline', 9),
(2, 'Я самостоятельно контролирую качество своей работы', 'external_control', 10);

-- Варианты ответов для ВСК (5-балльная прямая шкала)
INSERT INTO question_options (question_id, option_text, score_value, order_index) VALUES 
-- Вопрос 1 (настойчивость) - все прямые
(11, 'Полностью не согласен', 1, 1),
(11, 'Скорее не согласен', 2, 2),
(11, 'Затрудняюсь ответить', 3, 3),
(11, 'Скорее согласен', 4, 4),
(11, 'Полностью согласен', 5, 5),

-- Вопрос 2 (самомотивация) - все прямые
(12, 'Полностью не согласен', 1, 1),
(12, 'Скорее не согласен', 2, 2),
(12, 'Затрудняюсь ответить', 3, 3),
(12, 'Скорее согласен', 4, 4),
(12, 'Полностью согласен', 5, 5),

-- Вопрос 3 (прокрастинация) - теперь прямая шкала
(13, 'Полностью не согласен', 1, 1),
(13, 'Скорее не согласен', 2, 2),
(13, 'Затрудняюсь ответить', 3, 3),
(13, 'Скорее согласен', 4, 4),
(13, 'Полностью согласен', 5, 5),

-- Вопрос 4 (эмоциональный контроль) - прямая шкала
(14, 'Полностью не согласен', 1, 1),
(14, 'Скорее не согласен', 2, 2),
(14, 'Затрудняюсь ответить', 3, 3),
(14, 'Скорее согласен', 4, 4),
(14, 'Полностью согласен', 5, 5),

-- Вопрос 5 (концентрация) - прямая шкала
(15, 'Полностью не согласен', 1, 1),
(15, 'Скорее не согласен', 2, 2),
(15, 'Затрудняюсь ответить', 3, 3),
(15, 'Скорее согласен', 4, 4),
(15, 'Полностью согласен', 5, 5),

-- Вопрос 6 (дисциплина) - теперь прямая шкала
(16, 'Полностью не согласен', 1, 1),
(16, 'Скорее не согласен', 2, 2),
(16, 'Затрудняюсь ответить', 3, 3),
(16, 'Скорее согласен', 4, 4),
(16, 'Полностью согласен', 5, 5),

-- Вопрос 7 (ответственность) - прямая шкала
(17, 'Полностью не согласен', 1, 1),
(17, 'Скорее не согласен', 2, 2),
(17, 'Затрудняюсь ответить', 3, 3),
(17, 'Скорее согласен', 4, 4),
(17, 'Полностью согласен', 5, 5),

-- Вопрос 8 (устойчивость) - теперь прямая шкала
(18, 'Полностью не согласен', 1, 1),
(18, 'Скорее не согласен', 2, 2),
(18, 'Затрудняюсь ответить', 3, 3),
(18, 'Скорее согласен', 4, 4),
(18, 'Полностью согласен', 5, 5),

-- Вопрос 9 (самодисциплина) - прямая шкала
(19, 'Полностью не согласен', 1, 1),
(19, 'Скорее не согласен', 2, 2),
(19, 'Затрудняюсь ответить', 3, 3),
(19, 'Скорее согласен', 4, 4),
(19, 'Полностью согласен', 5, 5),

-- Вопрос 10 (внешний контроль) - теперь прямая шкала
(20, 'Полностью не согласен', 1, 1),
(20, 'Скорее не согласен', 2, 2),
(20, 'Затрудняюсь ответить', 3, 3),
(20, 'Скорее согласен', 4, 4),
(20, 'Полностью согласен', 5, 5);

-- Тест 3: Сокращенный вариант 16PF (ключевые факторы для ИБ)
INSERT INTO psychological_tests (title, description, instructions, estimated_time, pass_threshold, methodology_type) VALUES 
('Сокращенный опросник 16PF (ключевые факторы)',
'Оценка ключевых личностных факторов, важных для работы в информационной безопасности',
'Выберите один из двух предложенных вариантов ответа, который наиболее точно описывает ваше обычное поведение или предпочтения.',
25, 65.0, 'personality_16pf');

-- Вопросы для сокращенного 16PF (уже прямая шкала)
INSERT INTO test_questions (test_id, question_text, scale_type, order_index) VALUES 
(3, 'В незнакомой ситуации я обычно:', 'factor_A', 1),
(3, 'При принятии решений я больше полагаюсь:', 'factor_B', 2),
(3, 'В стрессовой ситуации я склонен:', 'factor_C', 3),
(3, 'При работе над проектом я предпочитаю:', 'factor_E', 4),
(3, 'В профессиональной деятельности я больше ценю:', 'factor_F', 5),
(3, 'При нарушении правил безопасности я:', 'factor_G', 6),
(3, 'В общении с коллегами я обычно:', 'factor_H', 7),
(3, 'При анализе информации я склонен:', 'factor_I', 8),
(3, 'В отношении новых технологий я:', 'factor_L', 9),
(3, 'При планировании работы я:', 'factor_Q3', 10);

-- Варианты ответов для 16PF (прямая шкала)
INSERT INTO question_options (question_id, option_text, score_value, order_index) VALUES 
-- Вопрос 1 (Фактор A: Замкнутость/Общительность) - 1 = замкнутость, 6 = общительность
(21, 'Действовать осторожно и осмотрительно', 1, 1),
(21, 'Легко идти на контакт с новыми людьми', 6, 2),

-- Вопрос 2 (Фактор B: Конкретное/Абстрактное мышление) - 1 = конкретное, 6 = абстрактное
(22, 'На проверенные факты и данные', 1, 1),
(22, 'На интуицию и общее впечатление', 6, 2),

-- Вопрос 3 (Фактор C: Эмоциональная нестабильность/Стабильность) - 1 = нестабильность, 6 = стабильность
(23, 'Волноваться и переживать', 1, 1),
(23, 'Сохранять спокойствие и хладнокровие', 6, 2),

-- Вопрос 4 (Фактор E: Подчиняемость/Доминантность) - 1 = подчиняемость, 6 = доминантность
(24, 'Следовать инструкциям и правилам', 1, 1),
(24, 'Проявлять инициативу и самостоятельность', 6, 2),

-- Вопрос 5 (Фактор F: Сдержанность/Беззаботность) - 1 = сдержанность, 6 = беззаботность
(25, 'Точность и аккуратность', 1, 1),
(25, 'Скорость и эффективность', 6, 2),

-- Вопрос 6 (Фактор G: Низкая/Высокая нормативность) - 1 = низкая, 6 = высокая
(26, 'Могу отступить от правил для достижения цели', 1, 1),
(26, 'Строго соблюдаю установленные процедуры', 6, 2),

-- Вопрос 7 (Фактор H: Робость/Смелость) - 1 = робость, 6 = смелость
(27, 'Соблюдаю дистанцию и формальности', 1, 1),
(27, 'Общаюсь открыто и неформально', 6, 2),

-- Вопрос 8 (Фактор I: Жесткость/Чувствительность) - 1 = жесткость, 6 = чувствительность
(28, 'Опираться на логику и факты', 1, 1),
(28, 'Учитывать человеческий фактор и обстоятельства', 6, 2),

-- Вопрос 9 (Фактор L: Доверчивость/Подозрительность) - 1 = доверчивость, 6 = подозрительность
(29, 'С интересом изучаю и тестирую новое', 1, 1),
(29, 'Отношусь с осторожностью и проверяю безопасность', 6, 2),

-- Вопрос 10 (Фактор Q3: Низкий/Высокий самоконтроль) - 1 = низкий, 6 = высокий
(30, 'Действую по ситуации, гибко подхожу к задачам', 1, 1),
(30, 'Ставлю четкие цели и следую плану', 6, 2);

-- Создаем тестовых пользователей (пароли будут установлены через функцию CreateTestUsers)
INSERT INTO users (email, password_hash, last_name, first_name, patronymic, role, is_blocked) VALUES 
('admin@psycho.test', 'temp_password', 'Администратор', 'Системы', '', 'admin', false),
('user@test.ru', 'temp_password', 'Пользователь', 'Тестовый', 'Тестович', 'user', false);