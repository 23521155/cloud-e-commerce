-- How a session was signed in (the access token's amr claim): "pwd" for a
-- password alone, "pwd otp" when an authenticator or recovery code followed.
-- Kept with the session so a refreshed token says the same.
ALTER TABLE refresh_tokens ADD
    amr NVARCHAR(20) NOT NULL CONSTRAINT df_refresh_tokens_amr DEFAULT N'pwd';
