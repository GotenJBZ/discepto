CREATE TABLE roles (
	id int primary key generated always as identity,
	name varchar(50) not null
);
INSERT INTO roles (name) VALUES ('admin');
CREATE TABLE users (
	id int primary key generated always as identity not null,
	name varchar(50) not null,
	email varchar(100) not null,
	role_id int references roles(id) not null default 0
);
CREATE TYPE permission_type AS ENUM (
	'add_mods',
	'delete_posts', -- post is every type of content
	'ban_users',
	'flag_posts'
);
CREATE TABLE role_permissions (
	role_id int references roles(id),
	permission permission_type,
	primary key(role_id, permission)
);
CREATE TABLE essays (
	id int primary key generated always as identity,
	thesis varchar(350) not null,
	content text not null,
	attributed_to_id int references users(id) not null,
	published timestamp not null
);
CREATE TABLE essay_mentions (
	essay_id int primary key references essays(id),
	mention_id int references essays(id) not null
);
CREATE TABLE essay_tags (
	essay_id int references essays(id),
	tag varchar(15),
	primary key(essay_id, tag)
);
CREATE TABLE essay_sources (
	essay_id int references essays(id),
	source varchar(255),
	primary key(essay_id, source)
);
CREATE TABLE subdisceptos (
	name varchar(50) primary key,
	description varchar(500) not null
);
CREATE TABLE subdiscepto_users (
	name varchar(50),
	user_id int references users(id),
	role_id int references roles(id),
	primary key(name, user_id)
);
CREATE TYPE flag_type as ENUM ('offensive', 'fake', 'spam', 'inaccurate');
CREATE TABLE reports (
	id int primary key generated always as identity,
	flag flag_type not null,
	description varchar(500),
	essay_id int references essays(id),
	from_user_id int references users(id) not null,
	to_user_id int references users(id) not null
);
CREATE TYPE vote_type as ENUM ('upvote', 'downvote');
CREATE TABLE votes (
	user_id int references users(id),
	essay_id int references essays(id),
	vote_type vote_type not null,
	primary key(user_id, essay_id)
);
