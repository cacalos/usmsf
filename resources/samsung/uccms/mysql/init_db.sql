CREATE TABLE IF NOT EXISTS conf_info (
  id               INT(11)         NOT NULL AUTO_INCREMENT,
  conf_id          VARCHAR(128)    NOT NULL,
  meta_id          VARCHAR(128)    NOT NULL,
  tag              VARCHAR(128),
  confdata         VARCHAR(4096),
  description      VARCHAR(256),
  timestamp        TIMESTAMP       NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (conf_id)
);

CREATE TABLE IF NOT EXISTS meta_info (
  id               INT(11)         NOT NULL AUTO_INCREMENT,
  meta_id          VARCHAR(128)    NOT NULL,
  service_name     VARCHAR(64)     NOT NULL,
  config_name      VARCHAR(64)     NOT NULL,
  metadata         VARCHAR(4096),
  useConfigMaps    BOOLEAN NOT NULL default false,
  validator_url    VARCHAR(1024)   NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (meta_id)
);

CREATE TABLE IF NOT EXISTS watch_info (
  id               INT(11)         NOT NULL AUTO_INCREMENT,
  watch_id         VARCHAR(128)    NOT NULL,
  conf_id          VARCHAR(128)    NOT NULL,
  call_back        VARCHAR(1024)   NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (watch_id)
);

CREATE TABLE IF NOT EXISTS snapshot (
  id              VARCHAR(48)     NOT NULL,
  config_id       VARCHAR(128)    NOT NULL,
  data            VARCHAR(4096)   NOT NULL,
  description     VARCHAR(256),
  status          VARCHAR(16)     NOT NULL,
  timestamp       TIMESTAMP       NOT NULL,
  PRIMARY KEY (id)
);
