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

CREATE TABLE subdiscepto_users (
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	PRIMARY KEY(subdiscepto, user_id)
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
	global_perms_id int REFERENCES global_perms(id) NOT NULL,
	sub_perms_id int REFERENCES sub_perms(id) NOT NULL,
	name varchar(50) NOT NULL,
	preset boolean NOT NULL,
	PRIMARY KEY (global_perms_id, sub_perms_id)
);

CREATE TABLE sub_roles (
	sub_perms_id int REFERENCES sub_perms(id),
	name varchar(50) NOT NULL,
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	preset boolean NOT NULL,
	PRIMARY KEY (sub_perms_id),
	UNIQUE(subdiscepto, preset, name)
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
(global_perms_id, sub_perms_id, name, preset)
VALUES
(-123, -123, 'admin', true);

INSERT INTO sub_roles
(sub_perms_id, subdiscepto, name, preset)
VALUES
(-123, NULL, 'admin', true),
(-100, NULL, 'moderator', true),
(-99,  NULL, 'judge', true);

CREATE TABLE tokens (
	token varchar(255),
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	last_used TIMESTAMP,
	last_used_ip inet,
	PRIMARY KEY(user_id, token)
);

CREATE TABLE user_global_roles (
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	global_perms_id int NOT NULL,
	sub_perms_id int NOT NULL,
	PRIMARY KEY(user_id, global_perms_id, sub_perms_id),
	FOREIGN KEY(global_perms_id, sub_perms_id) REFERENCES global_roles(global_perms_id, sub_perms_id) ON DELETE CASCADE
);

CREATE TABLE user_sub_roles (
	user_id int NOT NULL,
	subdiscepto varchar(50) NOT NULL,
	sub_perms_id int REFERENCES sub_roles(sub_perms_id) ON DELETE CASCADE NOT NULL,
	FOREIGN KEY (user_id, subdiscepto) REFERENCES subdiscepto_users(user_id, subdiscepto) ON DELETE CASCADE, 
	PRIMARY KEY(user_id, sub_perms_id, subdiscepto)
);

CREATE TABLE essays (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	thesis varchar(350) NOT NULL,
	content text NOT NULL,
	attributed_to_id int REFERENCES users(id) NOT NULL,
	posted_in varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE NOT NULL,
	published timestamp NOT NULL
);

CREATE TABLE essay_replies (
	from_id int REFERENCES essays(id) ON DELETE CASCADE,
	to_id int REFERENCES essays(id) ON DELETE CASCADE,
	reply_type varchar(24) NOT NULL,
	PRIMARY KEY (from_id, to_id)
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
	from_user_id int REFERENCES users(id) NOT NULL
);

CREATE TYPE vote_type as ENUM ('upvote', 'downvote');

CREATE TABLE votes (
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	vote_type vote_type NOT NULL,
	PRIMARY KEY(user_id, essay_id)
);
