CREATE TABLE "contract"
(
    "blockchain_name" VARCHAR(30)             NOT NULL,
    "address"         VARCHAR(100)            NOT NULL,
    "name"            VARCHAR(100) DEFAULT '' NOT NULL,
    "decimals"        INT          DEFAULT 0  NOT NULL,
    "symbol"          VARCHAR(50)  DEFAULT '' NOT NULL,
    "status"          VARCHAR(50)             NOT NULL,
    PRIMARY KEY ("blockchain_name", "address")
);
