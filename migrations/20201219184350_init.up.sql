CREATE TABLE users (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name varchar(50) NOT NULL,
	email varchar(100) UNIQUE NOT NULL,
	passwd_hash varchar(255) NOT NULL
);

CREATE TABLE subdisceptos (
	name varchar(50) PRIMARY KEY,
	description varchar(500) NOT NULL,
	min_length int NOT NULL,
	questions_required boolean NOT NULL,
	public boolean NOT NULL,
	nsfw boolean NOT NULL
);

CREATE TABLE subdiscepto_users (
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	left_at timestamp,
	PRIMARY KEY(subdiscepto, user_id)
);


-- Authorization:
-- There are multiple roles inside a domain
-- A role has multiple permissions
-- A user has multiple roles

CREATE TABLE roles (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	domain varchar(100) NOT NULL,
	name varchar(100) NOT NULL,
	preset boolean NOT NULL,
	UNIQUE (domain, name)
);

CREATE TABLE role_perms (
	role_id int REFERENCES roles(id) ON DELETE CASCADE,
	permission varchar(100) NOT NULL,
	PRIMARY KEY (role_id, permission)
);

CREATE TABLE user_roles (
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	role_id int REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

INSERT INTO roles (id, domain, name, preset)
OVERRIDING SYSTEM VALUE VALUES
(-123, 'discepto', 'admin', true),
(-100, 'discepto', 'common', false);

INSERT INTO role_perms (role_id, permission)
VALUES
-- admin
(-123, 'create_subdiscepto'),
(-123, 'read_subdiscepto'),
(-123, 'update_subdiscepto'),
(-123, 'delete_subdiscepto'),
(-123, 'ban_user_globally'),
(-123, 'manage_global_role'),
(-123, 'login'),
(-123, 'delete_user'),
(-123, 'create_essay'),
(-123, 'delete_essay'),
(-123, 'change_ranking'),
(-123, 'ban_user'),
(-123, 'manage_role'),
(-123, 'common_after_rejoin'),
(-123, 'view_report'),
(-123, 'delete_report'),
-- common
(-100, 'login');


CREATE TABLE tokens (
	token varchar(255),
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	last_used TIMESTAMP,
	last_used_ip inet,
	PRIMARY KEY(user_id, token)
);

CREATE TABLE essays (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	thesis varchar(350) NOT NULL,
	content text NOT NULL,
	attributed_to_id int NOT NULL REFERENCES users(id) ON DELETE SET NULL,
	posted_in varchar(50) NOT NULL REFERENCES subdisceptos(name) ON DELETE CASCADE,
	published timestamp NOT NULL
);

CREATE TABLE essay_replies (
	from_id int PRIMARY KEY REFERENCES essays(id) ON DELETE CASCADE,
	to_id int REFERENCES essays(id) ON DELETE CASCADE,
	reply_type varchar(24) NOT NULL
);

CREATE TABLE essay_tags (
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	tag varchar(15) NOT NULL,
	PRIMARY KEY(essay_id, tag)
);

CREATE TABLE essay_sources (
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	source varchar(255) NOT NULL,
	PRIMARY KEY(essay_id, source)
);

CREATE TABLE notifications (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	user_id int NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	notif_type varchar(30) NOT NULL,
	description varchar(500) NOT NULL,
	action_url varchar(255) NOT NULL
);

CREATE TABLE questions (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	essay_id int NOT NULL REFERENCES essays(id) ON DELETE CASCADE,
	text varchar(500) NOT NULL
);

CREATE TABLE answers (
	question_id int PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
	text varchar(250) NOT NULL,
	correct boolean NOT NULL
);

CREATE TYPE flag_type as ENUM ('offensive', 'fake', 'spam', 'inaccurate');

CREATE TABLE reports (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	description varchar(500),
	essay_id int REFERENCES essays(id) NOT NULL,
	from_user_id int REFERENCES users(id) NOT NULL,
	UNIQUE(essay_id, from_user_id)
);

CREATE TYPE vote_type as ENUM ('upvote', 'downvote');

CREATE TABLE votes (
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	vote_type vote_type NOT NULL,
	PRIMARY KEY(user_id, essay_id)
);
