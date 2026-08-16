CREATE TABLE users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL
);

CREATE TABLE refresh_tokens (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  family_id CHAR(36) NOT NULL,
  token_hash CHAR(64) NOT NULL UNIQUE,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  replaced_by_hash CHAR(64) NULL,
  created_at DATETIME(3) NOT NULL,
  INDEX idx_refresh_user_family (user_id, family_id),
  CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE interview_sessions (
  id CHAR(36) PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(24) NOT NULL,
  direction VARCHAR(128) NOT NULL DEFAULT '',
  resume_score INT NOT NULL DEFAULT 0,
  current_question_number VARCHAR(32) NOT NULL DEFAULT '',
  current_main_index INT NOT NULL DEFAULT 0,
  follow_up_count INT NOT NULL DEFAULT 0,
  total_score INT NOT NULL DEFAULT 0,
  turn_sequence BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  started_at DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  INDEX idx_session_user_created (user_id, created_at),
  CONSTRAINT fk_session_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE resume_assets (
  id CHAR(36) PRIMARY KEY,
  session_id CHAR(36) NOT NULL UNIQUE,
  original_name VARCHAR(255) NOT NULL,
  storage_key VARCHAR(512) NOT NULL,
  content_type VARCHAR(128) NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256 CHAR(64) NOT NULL,
  extracted_text MEDIUMTEXT NULL,
  created_at DATETIME(3) NOT NULL,
  CONSTRAINT fk_resume_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);

CREATE TABLE analysis_jobs (
  id CHAR(36) PRIMARY KEY,
  session_id CHAR(36) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(24) NOT NULL,
  progress INT NOT NULL DEFAULT 0,
  stage VARCHAR(64) NOT NULL DEFAULT '',
  prompt_version VARCHAR(32) NOT NULL DEFAULT 'v1',
  attempts INT NOT NULL DEFAULT 0,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  error_message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  INDEX idx_job_session (session_id),
  INDEX idx_job_user (user_id),
  CONSTRAINT fk_job_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);

CREATE TABLE interview_questions (
  id CHAR(36) PRIMARY KEY,
  session_id CHAR(36) NOT NULL,
  number VARCHAR(32) NOT NULL,
  content TEXT NOT NULL,
  suggestion TEXT NOT NULL,
  is_follow_up BOOLEAN NOT NULL DEFAULT FALSE,
  parent_number VARCHAR(32) NOT NULL DEFAULT '',
  ordinal INT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_question_session_number (session_id, number),
  CONSTRAINT fk_question_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);

CREATE TABLE answer_attempts (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id CHAR(36) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  question_number VARCHAR(32) NOT NULL,
  status VARCHAR(24) NOT NULL,
  lease_until DATETIME(3) NULL,
  response_json JSON NULL,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_attempt_session_key (session_id, idempotency_key),
  CONSTRAINT fk_attempt_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);

CREATE TABLE interview_turns (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id CHAR(36) NOT NULL,
  sequence BIGINT NOT NULL,
  request_id VARCHAR(128) NOT NULL,
  question_number VARCHAR(32) NOT NULL,
  question_content TEXT NOT NULL,
  answer_content TEXT NOT NULL,
  score INT NOT NULL,
  feedback TEXT NOT NULL,
  missing_points JSON NULL,
  is_follow_up BOOLEAN NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_turn_sequence (session_id, sequence),
  UNIQUE KEY uk_turn_request (session_id, request_id),
  CONSTRAINT fk_turn_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);

CREATE TABLE interview_reports (
  id CHAR(36) PRIMARY KEY,
  session_id CHAR(36) NOT NULL UNIQUE,
  user_id BIGINT UNSIGNED NOT NULL,
  interview_score INT NOT NULL,
  resume_score INT NOT NULL,
  composite_score INT NOT NULL,
  overall_comment TEXT NOT NULL,
  highlights JSON NULL,
  improvement_tips JSON NULL,
  next_actions JSON NULL,
  radar_metrics JSON NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  CONSTRAINT fk_report_session FOREIGN KEY (session_id) REFERENCES interview_sessions(id)
);
