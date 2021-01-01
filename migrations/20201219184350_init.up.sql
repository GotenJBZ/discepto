CREATE TABLE roles (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	permissions varchar(500) NOT NULL,
	name varchar(50) NOT NULL
);
-- Create initial roles. Manually set an easy to remember id.
INSERT INTO roles (id, name, permissions) OVERRIDING SYSTEM VALUE VALUES (
	-123, 
	'admin', 
	'delete_posts ban_users'
);
INSERT INTO roles (id, name, permissions) OVERRIDING SYSTEM VALUE VALUES (0, 'default', '');
CREATE TABLE users (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name varchar(50) NOT NULL,
	email varchar(100) UNIQUE NOT NULL,
	role_id int REFERENCES roles(id) DEFAULT 0
);
CREATE TABLE credentials (
	user_id int PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	hash varchar(255)
);
CREATE TABLE tokens (
	token varchar(255),
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, token)
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
