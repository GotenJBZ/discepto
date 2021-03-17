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
	nsfw boolean NOT NULL
);

CREATE TABLE global_perms (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	login boolean NOT NULL,
	create_subdiscepto boolean NOT NULL,
	hide_subdiscepto boolean NOT NULL,
	ban_user_globally boolean NOT NULL,
	delete_user boolean NOT NULL,
	add_admin boolean NOT NULL
);

CREATE TABLE subdiscepto_users (
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	user_id int REFERENCES users(id) ON DELETE CASCADE NOT NULL,
	PRIMARY KEY (subdiscepto, user_id)
);

CREATE TABLE sub_perms (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	delete_essay boolean NOT NULL,
	create_essay boolean NOT NULL,
	ban_user boolean NOT NULL,
	change_ranking boolean NOT NULL,
	delete_subdiscepto boolean NOT NULL,
	add_mod boolean NOT NULL
);

CREATE TABLE global_roles (
	name varchar(50) PRIMARY KEY,
	global_perms_id int REFERENCES global_perms(id) NOT NULL,
	sub_perms_id int REFERENCES sub_perms(id)
);

CREATE TABLE custom_sub_roles (
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE NOT NULL,
	name varchar(50) NOT NULL,
	sub_perms_id int REFERENCES sub_perms(id) NOT NULL,
	PRIMARY KEY (subdiscepto, name)
);

CREATE TABLE preset_sub_roles (
	name varchar(50) PRIMARY KEY,
	sub_perms_id int REFERENCES sub_perms(id) NOT NULL
);

-- Create initial roles. Manually set an easy to remember id.
INSERT INTO global_perms
(id,    login,  create_subdiscepto,  hide_subdiscepto,  ban_user_globally,  delete_user, add_admin)
OVERRIDING SYSTEM VALUE VALUES                                                                
(-123,  true,   true,                true,              true,               true,        true);

INSERT INTO sub_perms
(id,    create_essay,  delete_essay,  ban_user,  change_ranking,  delete_subdiscepto,  add_mod)
OVERRIDING SYSTEM VALUE VALUES
(-123,  true,          true,          true,      true,            true,                true),
(-100,  true,          true,          true,      true,            true,                false),
(-99,   true,          false,         false,     true,            false,               false);

INSERT INTO global_roles
(name,       global_perms_id,  sub_perms_id)
VALUES
('admin',    -123,             -123);

INSERT INTO preset_sub_roles
(name,        sub_perms_id)
VALUES
('admin',     -123),
('moderator', -100),
('judge',     -99);

CREATE TABLE tokens (
	token varchar(255),
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	last_used TIMESTAMP,
	last_used_ip inet,
	PRIMARY KEY(user_id, token)
);
CREATE TABLE user_custom_sub_roles (
	subdiscepto varchar(50),
	user_id int,
	role_name varchar(50),
	PRIMARY KEY(user_id, role_name, subdiscepto),
	FOREIGN KEY(subdiscepto, role_name) REFERENCES custom_sub_roles(subdiscepto, name) ON DELETE CASCADE,
	FOREIGN KEY(subdiscepto, user_id) REFERENCES subdiscepto_users(subdiscepto, user_id) ON DELETE CASCADE
);
CREATE TABLE user_preset_sub_roles (
	subdiscepto varchar(50),
	user_id int,
	role_name varchar(50) REFERENCES preset_sub_roles(name),
	PRIMARY KEY(user_id, subdiscepto, role_name),
	FOREIGN KEY(subdiscepto, user_id) REFERENCES subdiscepto_users(subdiscepto, user_id) ON DELETE CASCADE
);
CREATE TABLE user_global_roles (
	user_id int REFERENCES users(id) ON DELETE CASCADE NOT NULL,
	role_name varchar(50) REFERENCES global_roles(name) ON DELETE CASCADE NOT NULL,
	PRIMARY KEY(user_id, role_name)
);

CREATE TABLE essays (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	thesis varchar(350) NOT NULL,
	content text NOT NULL,
	attributed_to_id int REFERENCES users(id) NOT NULL,
	posted_in varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE NOT NULL,
	in_reply_to int REFERENCES essays(id),
	reply_type varchar(24) NOT NULL,
	published timestamp NOT NULL
);
CREATE TABLE essay_tags (
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	tag varchar(15),
	PRIMARY KEY(essay_id, tag)
);
CREATE TABLE essay_sources (
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	source varchar(255),
	PRIMARY KEY(essay_id, source)
);
CREATE TYPE flag_type as ENUM ('offensive', 'fake', 'spam', 'inaccurate');
CREATE TABLE reports (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	flag flag_type NOT NULL,
	description varchar(500),
	essay_id int REFERENCES essays(id),
	from_user_id int REFERENCES users(id) NOT NULL,
	to_user_id int REFERENCES users(id) NOT NULL
);
CREATE TYPE vote_type as ENUM ('upvote', 'downvote');
CREATE TABLE votes (
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	vote_type vote_type NOT NULL,
	PRIMARY KEY(user_id, essay_id)
);
