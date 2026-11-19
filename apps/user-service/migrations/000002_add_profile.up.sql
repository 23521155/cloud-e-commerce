-- Contact and delivery details the reader keeps on the account, used to fill
-- in checkout. Empty means not given; every order still keeps its own copy.
-- Lengths match the order service's checks, so a saved profile is accepted
-- at checkout.
ALTER TABLE users ADD
    phone       NVARCHAR(20)  NOT NULL CONSTRAINT df_users_phone DEFAULT N'',
    address     NVARCHAR(300) NOT NULL CONSTRAINT df_users_address DEFAULT N'',
    city        NVARCHAR(100) NOT NULL CONSTRAINT df_users_city DEFAULT N'',
    postal_code NVARCHAR(20)  NOT NULL CONSTRAINT df_users_postal_code DEFAULT N'',
    country     NVARCHAR(60)  NOT NULL CONSTRAINT df_users_country DEFAULT N'';
