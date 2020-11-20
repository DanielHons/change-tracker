INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('productive-service', 'alpha', 'newer-sha', 'sha', '2020-11-20 10:21:16.564772');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('productive-service', 'beta', '1.0.0', 'sha', '2020-11-20 10:54:15.063254');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('productive-service', 'prod', '1.0.0', 'sha', '2020-11-20 10:35:50.574065');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('productive-service', 'alpha', '1.1.0', 'very-new', '2020-11-20 10:20:51.293827');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('productive-service', 'develop', 'veryNew', 'even-newer', '2020-11-20 10:36:12.795538');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('undeployed-service', 'uds-develop', '829614', '829614', '2020-11-20 10:53:38.299571');
INSERT INTO data.stages_reached_latest_versions (component_name, stage_name, version, git_sha, timestamp)
VALUES ('undeployed-service', 'uds-develop', 'new', 'new', '2020-11-20 10:42:55.494138');


INSERT INTO data.stages_aliases (component_name, plain_stage, alias) VALUES ('undeployed-service', 'uds-develop', 'develop');