--testing auth
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_login', 'examplelogin@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', 'd1a5b8c9-4b72-4e81-ad3d-6b5c4f2e9a1b');
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_recovery_password', 'examplerecoverypassword@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', '4f3e2d1c-0b9a-4f8e-bd7c-6b5a4f3e2d1c');
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_recovery_user', 'examplerecoveryuser@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', '15e4d3c2-b1a0-4f9e-8d7c-6b5a4f3e2d1c');
--testing user
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
('2026-05-12', '2026-05-12', null, 'user_for_update', 'exampleupdate@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', 'f7b3a4c1-8d2e-4b9a-9e1c-5f6a7b8c9d0e');
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_get', 'exampleget@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', '7b3e1f4a-6d2c-4b8a-9e1c-5f6a7b8c9d0e');
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_delete', 'exampledelete@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', '5a1f4b3e-2c7d-491c-a3f5-6b2d8e1c9a4f');
INSERT INTO users (created_at, updated_at, deleted_at, name, email, password, user_uuid) VALUES
    ('2026-05-12', '2026-05-12', null, 'user_for_remove', 'exampleremove@gmail.com', '$2a$10$wCq6LLOuoRrZhfTM5W714.YXGhG3TmA33IE4Irs9UJfAdspns6E2i', '9f8e7d6c-5b4a-4321-a1b2-c3d4e5f6a7b8');