CREATE TABLE media_tts_tasks (
  id CHAR(36) PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  provider_task_id VARCHAR(128) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  sid VARCHAR(128) NULL,
  task_status VARCHAR(16) NOT NULL,
  provider_code INT NOT NULL DEFAULT 0,
  provider_message VARCHAR(512) NULL,
  audio_encoding VARCHAR(16) NOT NULL,
  sample_rate INT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  completed_at DATETIME(3) NULL,
  CONSTRAINT fk_media_tts_tasks_user FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE KEY uk_media_tts_user_idempotency (user_id, idempotency_key),
  UNIQUE KEY uk_media_tts_provider_task (provider_task_id),
  KEY idx_media_tts_user_created (user_id, created_at)
);
