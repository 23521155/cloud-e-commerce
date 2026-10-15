ALTER TABLE users DROP CONSTRAINT df_users_phone, df_users_address, df_users_city, df_users_postal_code, df_users_country;
ALTER TABLE users DROP COLUMN phone, address, city, postal_code, country;
