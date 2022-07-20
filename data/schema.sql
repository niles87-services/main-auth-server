USE auth_db;

DROP TABLE IF EXISTS user;

CREATE TABLE
    user (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(200) NOT NULL,
        email VARCHAR(200) NOT NULL,
        password VARCHAR(255) NOT NULL
    );