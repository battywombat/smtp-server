
CREATE TABLE IF NOT EXISTS  address (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    domain TEXT
);

CREATE TABLE IF NOT EXISTS emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_addr INTEGER,
    to_addr INTEGER,
    subject TEXT,
    body TEXT,
    FOREIGN KEY(from_addr) REFERENCES address(id),
    FOREIGN KEY(to_addr) REFERENCES address(id)
);