-- Remove exactly the three rows the up migration may have inserted.
DELETE FROM carepath.user_role WHERE user_id = 'user-exec' AND role_id = 'role-executive';
DELETE FROM carepath.app_user WHERE user_id = 'user-exec';
DELETE FROM carepath.role WHERE role_id = 'role-executive';
