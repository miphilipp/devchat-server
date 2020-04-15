--DROP DATABASE devchat;
CREATE DATABASE devchat
    WITH 
    OWNER = default
    ENCODING = 'UTF8'
    LC_COLLATE = 'de_DE.UTF-8'
    LC_CTYPE = 'de_DE.UTF-8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

    \connect devchat

-- DROP TABLE public."user";
CREATE TABLE public."user" (
    name character varying(40) NOT NULL,
    email character varying(40) NOT NULL,
    password text NOT NULL,
    confirmation_uuid uuid,
    id SERIAL PRIMARY KEY,
    isonline boolean NOT NULL DEFAULT false,
    lastonline timestamp without time zone,
    failedloginattempts smallint NOT NULL DEFAULT 0,
    lockedoutsince timestamp without time zone,
    lastfailedlogin timestamp without time zone,
    isdeleted boolean NOT NULL DEFAULT false,
    recovery_uuid uuid,
    recovery_uuid_issue_date timestamp without time zone,
    CONSTRAINT user_email_key UNIQUE (email),
    CONSTRAINT user_name_key UNIQUE (name)
);

-- DROP INDEX public.lower_name_idx;
CREATE INDEX lower_name_idx ON public."user" USING btree
    (lower(name::text) COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;

-- DROP TABLE public.conversation;
CREATE TABLE public.conversation (
    title character varying(100) NOT NULL,
    repourl text,
    id SERIAL PRIMARY KEY
);


-- DROP TABLE public.group_association;
CREATE TABLE public.group_association (
    isadmin boolean NOT NULL DEFAULT false,
    userid integer REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    conversationid integer REFERENCES public.conversation MATCH SIMPLE ON DELETE CASCADE,
    joined timestamp without time zone,
    colorindex integer NOT NULL DEFAULT -1,
    hasleft boolean NOT NULL DEFAULT false,
    CONSTRAINT group_association_pkey PRIMARY KEY (userid, conversationid)
);

-- DROP TABLE public.message;
CREATE TABLE public.message (
    sentdate timestamp without time zone NOT NULL,
    conversationid integer NOT NULL REFERENCES public.conversation MATCH SIMPLE ON DELETE CASCADE,
    id BIGSERIAL PRIMARY KEY,
    userid integer NOT NULL REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    type integer NOT NULL
);

-- DROP INDEX public.message_conversationid_idx;
CREATE INDEX message_conversationid_idx ON public.message USING btree
    (conversationid ASC NULLS LAST)
    TABLESPACE pg_default;

-- DROP INDEX public.message_userid_idx;
CREATE INDEX message_userid_idx ON public.message USING btree
    (userid ASC NULLS LAST)
    TABLESPACE pg_default;


-- DROP TABLE public.programming_language;
CREATE TABLE public.programming_language (
    name character varying(20) PRIMARY KEY,
    runnable boolean NOT NULL
);


-- DROP TABLE public.code_message;
CREATE TABLE public.code_message (
    id BIGINT PRIMARY KEY REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    language character varying(20) NOT NULL REFERENCES public.programming_language (name) MATCH SIMPLE,
    code text NOT NULL,
    title character varying(40) NOT NULL,
    lockedby bigint REFERENCES public.user (id) on delete set null MATCH SIMPLE      
);

-- DROP INDEX public.code_message_language_idx;
CREATE INDEX code_message_language_idx
    ON public.code_message USING btree
    (language COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;


-- DROP TABLE public.text_message;
CREATE TABLE public.text_message (
    id BIGINT PRIMARY KEY REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    text text NOT NULL
);

CREATE OR REPLACE public.v_text_message AS
SELECT m.id, t.text, m.sentdate, m.conversationid, m.userid, m.type
FROM public.message m
JOIN public.text_message t ON m.id = t.id;

CREATE OR REPLACE public.v_code_message AS
SELECT m.id, c.code, m.sentdate, m.conversationid, m.userid, m.type, c.title, c.language, c.lockedby
FROM public.message m
JOIN public.code_message c ON m.id = c.id;

-- DROP TABLE public.message_status;
CREATE TABLE public.message_status (
    userid integer REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    messageid bigint REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    conversationid integer NOT NULL REFERENCES public.conversation (id) MATCH SIMPLE ON DELETE CASCADE,
    hasread boolean NOT NULL DEFAULT false,
    CONSTRAINT message_status_pkey PRIMARY KEY (messageid, userid)
);


CREATE OR REPLACE VIEW public.v_invitation
 AS
 SELECT c.id AS conversationid,
    c.title AS conversationtitle,
    g.userid AS recipient
   FROM group_association g
     JOIN conversation c ON c.id = g.conversationid
  WHERE g.joined IS NULL;

CREATE OR REPLACE VIEW public.v_conversation
 AS
 SELECT c.id,
    c.title,
    c.repourl,
    s.unreadmessagescount,
    g.userid
   FROM group_association g
     JOIN conversation c ON c.id = g.conversationid
     LEFT JOIN ( SELECT count(*) AS unreadmessagescount,
            message_status.conversationid,
            message_status.userid
           FROM message_status
          WHERE message_status.hasread = false
          GROUP BY message_status.conversationid, message_status.userid) s ON s.conversationid = c.id AND s.userid = g.userid
  WHERE g.joined IS NOT NULL;

CREATE OR REPLACE VIEW public.v_joined_member
 AS
 SELECT g.isadmin,
    g.userid,
    g.conversationid,
    g.colorindex,
    g.joined
   FROM group_association g
     JOIN "user" u ON u.id = g.userid
  WHERE g.hasleft = false AND g.joined IS NOT NULL AND u.isdeleted = false;

CREATE OR REPLACE VIEW public.v_admin
 AS
 SELECT group_association.isadmin,
    group_association.userid,
    group_association.conversationid,
    group_association.joined,
    group_association.colorindex
   FROM group_association
  WHERE group_association.isadmin = true AND group_association.joined IS NOT NULL;

CREATE OR REPLACE VIEW public.v_every_member
AS
 SELECT u.name,
    g.userid AS id,
    g.isadmin,
    g.colorindex,
    g.joined IS NOT NULL AS hasjoined,
    g.hasleft,
    u.isdeleted,
    g.conversationid
   FROM group_association g
     JOIN "user" u ON u.id = g.userid;


CREATE EXTENSION pgcrypto;
CREATE EXTENSION "uuid-ossp";