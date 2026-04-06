create table users
(
    id                 int auto_increment
        primary key,
    username           varchar(255) null,
    password           varchar(255) null,
    date_registered    datetime     null,
    avatar             varchar(255) null,
    date_last_visit    datetime     null,
    interface_language varchar(50)  null,
    interface_timezone varchar(50)  null,
    interface_font_size decimal(5,2) default 1.00 not null,
    user_status        int default 0 not null,
    total_posts        int default 0 not null,
    total_general_posts int default 0 not null,
    disable_sound      boolean default false not null,
    constraint users_pk_2
        unique (username)
);

INSERT INTO users (username, password, date_registered, avatar, date_last_visit, interface_language, interface_timezone, user_status, interface_font_size) VALUES ('guest', null, null, null, null, null, null, 0, 1.00);
UPDATE users SET id = 0 WHERE username = 'guest';
ALTER TABLE users AUTO_INCREMENT = 1;

INSERT INTO users (id, username, password, date_registered, avatar, date_last_visit, interface_language, interface_timezone, user_status, interface_font_size) VALUES (1, 'System', null, null, null, null, null, null, 0, 1.00);

create table user_role
(
    user_id int not null,
    role_id int not null,
    constraint user_role_pk
        primary key (user_id, role_id)
);

CREATE TABLE custom_field_config
(
    entity_type VARCHAR(255) NOT NULL,
    config      JSON         NULL,
    PRIMARY KEY (entity_type)
);

-- Indexing remains the same syntax
CREATE INDEX custom_field_config_entity_type_index
    ON custom_field_config (entity_type);

INSERT INTO custom_field_config (entity_type, config)
VALUES ('character', '[]')
    ON DUPLICATE KEY UPDATE config = '[]';

INSERT INTO custom_field_config (entity_type, config)
VALUES ('episode', '[]')
    ON DUPLICATE KEY UPDATE config = '[]';

INSERT INTO custom_field_config (entity_type, config)
VALUES ('character_profile', '[]')
    ON DUPLICATE KEY UPDATE config = '[]';

INSERT INTO custom_field_config (entity_type, config)
VALUES ('wanted_character', '[]')
    ON DUPLICATE KEY UPDATE config = '[]';


CREATE TABLE global_settings
(
    setting_name  VARCHAR(255) NOT NULL,
    setting_value VARCHAR(255),
    PRIMARY KEY (setting_name)
);

INSERT INTO global_settings (setting_name, setting_value)
VALUES ('site_name', 'Site Name');

INSERT INTO global_settings (setting_name, setting_value)
VALUES ('posts_per_page', '20');

INSERT INTO global_settings (setting_name, setting_value)
VALUES ('imgbb_api_key', '');

INSERT INTO global_settings (setting_name, setting_value)
VALUES ('allow_add_faction', 'y');

INSERT INTO global_settings (setting_name, setting_value)
VALUES ('allow_wanted_for_claims', 'moderated');

CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NULL,
    position INT NULL
);

CREATE TABLE subforums (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    category_id INT NULL,
    name VARCHAR(255) NULL,
    description TINYTEXT NULL,
    position INT NULL,
    topic_number INT NULL,
    post_number INT NULL,
    date_last_post DATETIME,
    last_post_topic_id bigint unsigned null,
    last_post_topic_name varchar(255) null,
    last_post_id bigint unsigned null,
    last_post_author_user_name varchar(255) null,
    show_last_topic boolean null,
    constraint subforums_categories_id_fk
        foreign key (category_id) references categories (id)
);

CREATE TABLE topics (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    status INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    type INT NOT NULL,
    date_created DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_post_id BIGINT UNSIGNED NULL,
    date_last_post DATETIME,
    last_post_author_user_id INT NULL,
    post_number INT,
    author_user_id INT NOT NULL,
    subforum_id BIGINT UNSIGNED NOT NULL,
    CONSTRAINT fk_topics_subforum
        FOREIGN KEY (subforum_id) REFERENCES subforums (id) ON DELETE NO ACTION ,
    CONSTRAINT fk_topics_user
        FOREIGN KEY (author_user_id) REFERENCES users (id) ON DELETE NO ACTION,
    CONSTRAINT fk_topics_last_post_user
        FOREIGN KEY (last_post_author_user_id) REFERENCES users (id) ON DELETE NO ACTION
);

create table character_base
		(id      bigint unsigned auto_increment primary key,
		user_id int          null,
		name    varchar(255) null,
		avatar  varchar(255) null,
        topic_id bigint unsigned null,
        character_status int default 2 not null,
        total_posts int default 0 null,
        total_episodes int default 0 null,
        date_last_post datetime null,
        is_archived boolean default false null,
		constraint character_base_users_id_fk
		foreign key (user_id) references users (id),
        constraint character_base_topics_id_fk
		foreign key (topic_id) references topics (id)
		);

create table character_main
(
    entity_id          int            null,
    field_machine_name varchar(255)   null,
    field_type         varchar(10)    null,
    value_int          int            null,
    value_decimal      decimal(10, 2) null,
    value_string       varchar(255)   null,
    value_text         text           null,
    value_date         varchar(255)   null
);

create table character_profile_base
		(id      bigint unsigned auto_increment primary key,
		character_id bigint unsigned          null,
        avatar varchar(255) null,
        is_archived boolean null,
        is_mask boolean null,
        mask_name varchar(255) null,
        user_id int null,
		constraint character_profile_base_character_id_fk
		foreign key (character_id) references character_base (id)  ON DELETE CASCADE
		);

create table character_profile_main
(
    entity_id          int            null,
    field_machine_name varchar(255)   null,
    field_type         varchar(10)    null,
    value_int          int            null,
    value_decimal      decimal(10, 2) null,
    value_string       varchar(255)   null,
    value_text         text           null,
    value_date         varchar(255)   null
);

CREATE TABLE posts (
                       id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
                       topic_id BIGINT UNSIGNED NOT NULL,
                       author_user_id INT NOT NULL,
                       guest_name VARCHAR(255) NULL,
                       date_created DATETIME DEFAULT CURRENT_TIMESTAMP,
                       content TEXT NOT NULL,
                       character_profile_id BIGINT UNSIGNED,
                       use_character_profile BOOLEAN DEFAULT FALSE,
                       is_deleted TINYINT NULL,
                       CONSTRAINT fk_posts_topic
                           FOREIGN KEY (topic_id) REFERENCES topics (id) ON DELETE CASCADE,
                       CONSTRAINT fk_posts_user
                           FOREIGN KEY (author_user_id) REFERENCES users (id) ON DELETE CASCADE,
                       CONSTRAINT fk_posts_character_profile
                           FOREIGN KEY (character_profile_id) REFERENCES character_profile_base (id) ON DELETE SET NULL
);

ALTER TABLE topics ADD CONSTRAINT topics_posts_id_fk FOREIGN KEY (last_post_id) REFERENCES posts (id);

create table episode_base
		(id               bigint unsigned auto_increment primary key,
		topic_id          bigint unsigned null,
		name              varchar(255)    null,
		open_to_everyone  boolean         default false not null,
		constraint episode_base_topics_id_fk
		foreign key (topic_id) references topics (id)
		);

create table episode_main
(
    entity_id          int            null,
    field_machine_name varchar(255)   null,
    field_type         varchar(10)    null,
    value_int          int            null,
    value_decimal      decimal(10, 2) null,
    value_string       varchar(255)   null,
    value_text         text           null,
    value_date         varchar(255)   null
);

create table episode_character
		(episode_id bigint unsigned          null,
		character_id bigint unsigned          null,
         foreign key (episode_id) references episode_base (id),
         foreign key (character_id) references character_base (id)
		);

create table episode_mask
		(episode_id bigint unsigned          null,
		mask_id bigint unsigned          null,
         foreign key (episode_id) references episode_base (id),
         foreign key (mask_id) references character_profile_base (id)
		);

create table global_stats
(
    stat_name   varchar(255) null
        primary key,
    stat_value decimal      null,
    stat_secondary varchar(255) null
);

INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_user_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_character_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_active_character_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_episode_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_topic_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_post_number', 0);
INSERT INTO global_stats (stat_name, stat_value) VALUES ('total_episode_post_number', 0);
INSERT INTO global_stats (stat_name, stat_value, stat_secondary) VALUES ('last_user', 0, '');

create table factions
(
    id              int          auto_increment  primary key,
    name            varchar(255) not null,
    parent_id       int          null,
    level           int          not null,
    description     text         null,
    icon            varchar(255) null,
    show_on_profile boolean      not null,
    can_be_multiple bool default FALSE null,
    root_id         int          null,
    faction_status int default 2 not null
);

create table character_faction
(
    character_id bigint unsigned null,
    faction_id   int             null,
    constraint character_faction_character_base_id_fk
        foreign key (character_id) references character_base (id),
    constraint character_faction_factions_id_fk
        foreign key (faction_id) references factions (id)
);

create table character_flattened
(
    entity_id int primary key
);

create table character_profile_flattened
(
    entity_id int primary key
);

create table episode_flattened
(
    entity_id int primary key
);

create table roles
(
    id   int auto_increment
        primary key,
    name varchar(255) null
);

INSERT INTO roles (id, name) VALUES (1, 'guest');

INSERT INTO roles (id, name) VALUES (2, 'user');

INSERT INTO roles (id, name) VALUES (3, 'admin');

create table role_permission
(
    role_id    int          null,
    type       int          default 0,
    permission varchar(255) null,
    constraint role_permission_pk
        primary key (role_id, permission),
    constraint role_permission_roles_id_fk
        foreign key (role_id) references roles (id)
);

-- Default permissions for 'user' role (ID 2)
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 2, '/notifications/unread');
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 2, '/notifications/dismiss/:id');

-- Default permissions for 'admin' role (ID 3)
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 3, '/permission-matrix/get');
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 3, '/permission-matrix/update');
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 3, '/template/:type/update');
INSERT INTO role_permission (type, role_id, permission) VALUES (0, 3, '/character/accept/:id');

CREATE TABLE notifications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data JSON NULL,
    date_created DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_read BOOLEAN DEFAULT FALSE,
    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

create table recovery_codes
(
    id            int auto_increment primary key,
    user_id       int          not null,
    recovery_code varchar(255) not null,
    security_code varchar(64)  null,
    date_used     datetime     null,
    constraint fk_recovery_codes_user foreign key (user_id) references users (id) on delete cascade
);

create table private_keys
(
    id               int auto_increment primary key,
    user_id          int          not null,
    private_key      varchar(2000) not null,
    salt             varchar(255) not null,
    iv               varchar(255) not null,
    recovery_code_id int          null,
    is_active        boolean      not null default false,
    constraint fk_private_keys_user foreign key (user_id) references users (id) on delete cascade,
    constraint fk_private_keys_recovery_code foreign key (recovery_code_id) references recovery_codes (id) on delete set null
);

create table direct_chats
(
    id         int auto_increment primary key,
    start_date datetime not null,
    status     int      not null default 0
);

create table direct_chat_users
(
    direct_chat_id      int not null,
    user_id             int not null,
    last_read_message_id int null,
    unread_count         int not null default 0,
    constraint direct_chat_users_pk primary key (direct_chat_id, user_id),
    constraint fk_direct_chat_users_chat foreign key (direct_chat_id) references direct_chats (id) on delete cascade,
    constraint fk_direct_chat_users_user foreign key (user_id) references users (id) on delete cascade,
    constraint fk_direct_chat_users_last_message foreign key (last_read_message_id) references direct_chat_messages (id) on delete set null
);

create table direct_chat_messages
(
    id            int auto_increment primary key,
    chat_id       int      not null,
    user_id       int      not null,
    date_send     datetime not null,
    date_received datetime null,
    ciphertext       text not null,
    iv               varchar(24) not null,
    key_author       text not null,
    key_receiver     text not null,
    constraint fk_direct_chat_messages_chat foreign key (chat_id) references direct_chats (id) on delete cascade,
    constraint fk_direct_chat_messages_user foreign key (user_id) references users (id) on delete cascade
);

create table public_keys
(
    id         int auto_increment primary key,
    user_id    int          not null,
    public_key varchar(512) not null,
    constraint fk_public_keys_user foreign key (user_id) references users (id) on delete cascade
);

create table images
(
    id            int auto_increment primary key,
    url           varchar(512) not null,
    thumbnail_url varchar(512) null,
    user_id       int          not null,
    date_created  datetime     default current_timestamp,
    delete_url    varchar(512) null,
    constraint fk_images_user foreign key (user_id) references users (id) on delete cascade
);

create table user_topic_view
(
    user_id   int             not null,
    topic_id  bigint unsigned not null,
    post_id   bigint unsigned null,
    view_date datetime        default current_timestamp,
    primary key (user_id, topic_id),
    constraint fk_user_topic_view_user foreign key (user_id) references users (id) on delete cascade,
    constraint fk_user_topic_view_topic foreign key (topic_id) references topics (id) on delete cascade
);

create table character_claim
(
    id              int auto_increment primary key,
    name            varchar(255) not null,
    description     text         null,
    is_claimed      boolean      default false not null,
    user_id         int          null,
    guest_hash      varchar(255) null,
    can_change_name boolean      default false not null,
    last_claim_date datetime     null,
    constraint fk_character_claim_user foreign key (user_id) references users (id) on delete set null
);

create table character_claim_faction
(
    character_claim_id int null,
    faction_id         int null,
    constraint character_claim_faction_claim_id_fk
        foreign key (character_claim_id) references character_claim (id),
    constraint character_claim_faction_factions_id_fk
        foreign key (faction_id) references factions (id)
);

create table wanted_character_base
(
    id                 int auto_increment primary key,
    name               varchar(255) not null,
    is_claimed         boolean      default false not null,
    author_user_id     int          not null,
    date_created       datetime     default current_timestamp,
    character_claim_id int             null,
    is_deleted         boolean         null,
    topic_id           bigint unsigned null,
    constraint fk_wanted_character_author foreign key (author_user_id) references users (id) on delete cascade,
    constraint fk_wanted_character_claim  foreign key (character_claim_id) references character_claim (id) on delete set null,
    constraint fk_wanted_character_topic  foreign key (topic_id) references topics (id) on delete set null
);

create table wanted_character_main
(
    entity_id          int            null,
    field_machine_name varchar(255)   null,
    field_type         varchar(10)    null,
    value_int          int            null,
    value_decimal      decimal(10, 2) null,
    value_string       varchar(255)   null,
    value_text         text           null,
    value_date         varchar(255)   null
);

create table wanted_character_flattened
(
    entity_id int primary key
);

create table wanted_character_faction
(
    wanted_character_id int null,
    faction_id          int null,
    constraint wanted_character_faction_wanted_character_base_id_fk
        foreign key (wanted_character_id) references wanted_character_base (id),
    constraint wanted_character_faction_factions_id_fk
        foreign key (faction_id) references factions (id)
);

create table widget_types
(
    id              int auto_increment primary key,
    name            varchar(255) not null,
    config_template text         null,
    func            varchar(255) not null
);

INSERT INTO widget_types (name, config_template, func) VALUES ('last_post', '{"topic_id": {"type": "int"}}', 'WidgetLastPost');
INSERT INTO widget_types (name, config_template, func) VALUES ('random_entities', '{"number": {"type": "int"}, "entity_type": {"type": "string", "values": ["wanted_character", "character"]}, "entity_field_1": {"type": "string", "endpoint": "entity/fields/:entity_type", "can_empty": true}, "entity_field_2": {"type": "string", "endpoint": "entity/fields/:entity_type", "can_empty": true}}', 'WidgetRandomEntities');

create table widgets
(
    id          int auto_increment primary key,
    name        varchar(255) not null,
    template_id int          not null,
    config      text         null,
    constraint widgets_widget_types_id_fk
        foreign key (template_id) references widget_types (id)
);

create table widget_panels
(
    `key`     varchar(255) not null primary key,
    content   text         null,
    is_hidden boolean      not null default false
);

INSERT INTO widget_panels (`key`, content, is_hidden) VALUES ('header', NULL, false);

CREATE TABLE static_files
(
    file_name         varchar(255) not null primary key,
    file_created_date datetime     not null,
    file_type         varchar(255) null
);

INSERT INTO static_files (file_name, file_created_date, file_type) VALUES ('favicon.ico', '2026-03-01 00:00:00', 'favicon.ico');
INSERT INTO static_files (file_name, file_created_date, file_type) VALUES ('custom_style.css', '2026-03-31 00:00:00', 'custom_style.css');
INSERT INTO static_files (file_name, file_created_date, file_type) VALUES ('main_style.css', '2026-03-01 00:00:00', 'main_style.css');

CREATE TABLE design_variations
(
    id         int auto_increment primary key,
    class_name varchar(255) null,
    name       varchar(255) null
);