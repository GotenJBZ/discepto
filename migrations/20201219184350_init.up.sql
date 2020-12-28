CREATE TABLE roles (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name varchar(50) NOT NULL
);
-- Create initial roles. Manually set an easy to remember id.
INSERT INTO roles (id, name) OVERRIDING SYSTEM VALUE VALUES (-123, 'admin');
INSERT INTO roles (id, name) OVERRIDING SYSTEM VALUE VALUES (0, 'default');
CREATE TABLE users (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name varchar(50) NOT NULL,
	email varchar(100) UNIQUE NOT NULL,
	role_id int REFERENCES roles(id) DEFAULT 0
);
CREATE TYPE permission_type AS ENUM (
	'add_mods',
	'delete_posts', -- post is every type of content
	'ban_users',
	'flag_posts'
);
CREATE TABLE role_permissions (
	role_id int REFERENCES roles(id),
	permission permission_type,
	PRIMARY KEY(role_id, permission)
);
CREATE TABLE essays (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	thesis varchar(350) NOT NULL,
	content text NOT NULL,
	attributed_to_id int REFERENCES users(id) NOT NULL,
	published timestamp NOT NULL
);
CREATE TABLE essay_mentions (
	essay_id int REFERENCES essays(id) ON DELETE CASCADE,
	mention_id int REFERENCES essays(id) ON DELETE CASCADE,
	PRIMARY KEY(essay_id, mention_id)
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
CREATE TABLE subdisceptos (
	name varchar(50) PRIMARY KEY,
	description varchar(500) NOT NULL
);
CREATE TABLE subdiscepto_users (
	name varchar(50),
	user_id int REFERENCES users(id) ON DELETE CASCADE NOT NULL,
	role_id int REFERENCES roles(id) DEFAULT 0,
	PRIMARY KEY(name, user_id)
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
	user_id int REFERENCES users(id),
	essay_id int REFERENCES essays(id),
	vote_type vote_type NOT NULL,
	PRIMARY KEY(user_id, essay_id)
);
