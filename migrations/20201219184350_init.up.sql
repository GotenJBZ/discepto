CREATE TABLE users (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name varchar(50) NOT NULL,
	email varchar(100) UNIQUE NOT NULL,
	passwd_hash varchar(255)
);
CREATE TABLE roles (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	permissions varchar(500) NOT NULL,
	name varchar(50) NOT NULL,
	origin int REFERENCES users(id) NULL,
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create initial roles. Manually set an easy to remember id.
INSERT INTO roles (id, name, permissions) OVERRIDING SYSTEM VALUE VALUES (
	-123, 
	'admin', 
	'delete_posts ban_users'
),
(0, 'default', '');
CREATE TABLE tokens (
	token varchar(255),
	user_id int REFERENCES users(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, token)
);
CREATE TABLE subdisceptos (
	name varchar(50) PRIMARY KEY,
	description varchar(500) NOT NULL,
	min_length int NOT NULL,
	questions_required boolean NOT NULL,
	nsfw boolean NOT NULL
);
CREATE TABLE user_roles (
	user_id int REFERENCES users(id) ON DELETE CASCADE NOT NULL,
	role_id int REFERENCES roles(id) NOT NULL,
	subdiscepto varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	PRIMARY KEY(user_id, role_id, subdiscepto)
);
CREATE TABLE subdiscepto_users (
	name varchar(50) REFERENCES subdisceptos(name) ON DELETE CASCADE,
	user_id int REFERENCES users(id) ON DELETE CASCADE NOT NULL,
	PRIMARY KEY(name, user_id)
);
CREATE TABLE essays (
	id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	thesis varchar(350) NOT NULL,
	content text NOT NULL,
	attributed_to_id int REFERENCES users(id) NOT NULL,
	posted_in varchar(50) REFERENCES subdisceptos(name) NOT NULL,
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
	user_id int REFERENCES users(id),
	essay_id int REFERENCES essays(id),
	vote_type vote_type NOT NULL,
	PRIMARY KEY(user_id, essay_id)
);
