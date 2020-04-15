create or replace function createTextMessage(
    in v_userid integer, 
    in v_conversationId integer, 
    in v_sentDate timestamp,
    in v_text text)
RETURNS group_association.userid%TYPE 
AS $$
DECLARE newMessageId group_association.userid%TYPE;
DECLARE member RECORD;
begin
    
  INSERT INTO message (userid, conversationId, sentDate, type) 
  VALUES (v_userid, v_conversationId, v_sentDate, 0) 
  RETURNING id INTO newMessageId;

  INSERT INTO public.text_message (id, text) 
  VALUES (currval('message_id_seq'), v_text);

  INSERT INTO message_status(userid, messageid, conversationid, hasread)
  VALUES (v_userid, newMessageId, v_conversationId, true);

  FOR member IN (
      SELECT userid
      FROM public.group_association 
      WHERE userid != v_userid AND conversationid = v_conversationId 
  )
  LOOP
    INSERT INTO message_status(userid, messageid, conversationid, hasread)
    VALUES (member.userid, newMessageId, v_conversationId, false);
  END LOOP;
  
  RETURN newMessageId;
end;
$$ language PLpgSQL;


create or replace function createCodeMessage(
    in v_userid integer,
    in v_conversationId integer,
    in v_code text,
    in v_sentDate timestamp,
    in v_language varchar(20),
    in v_title varchar(40))
RETURNS group_association.userid%TYPE 
AS $$
DECLARE newMessageId group_association.userid%TYPE;
DECLARE member RECORD;
begin
    
  INSERT INTO message (userid, conversationId, sentDate, type) 
  VALUES (v_userid, v_conversationId, v_sentDate, 1) 
  RETURNING id INTO newMessageId;

  INSERT INTO public.code_message (id, code, title, language) 
  VALUES (currval('message_id_seq'), v_code, v_title, v_language);

  INSERT INTO message_status(userid, messageid, conversationid, hasread)
  VALUES (v_userid, newMessageId, v_conversationId, true);

  FOR member IN (
      SELECT userid
      FROM public.group_association 
      WHERE userid != v_userid AND conversationid = v_conversationId 
  )
  LOOP
    INSERT INTO message_status(userid, messageid, conversationid, hasread)
    VALUES (member.userid, newMessageId, v_conversationId, false);
  END LOOP;
  
  RETURN newMessageId;
end;
$$ language PLpgSQL;


create or replace function joinConversation(
    in v_userid integer,
    in v_conversationId integer)
RETURNS integer
AS $$
DECLARE v_hasleft Boolean default false;
DECLARE v_colorIndex INTEGER;
DECLARE v_res_colorIndex integer;
DECLARE v_exists Boolean;
begin
  select exists(
    select 1 from group_association 
    where conversationId = v_conversationId and userid = v_userid
  ) into v_exists

  if v_exists = false THEN
    RAISE EXCEPTION 'Not invited'
  end if;

  select hasleft 
  from group_association
  where userid = v_userid and conversationid = v_conversationId
  into v_hasleft;

  IF v_hasleft = true THEN
    UPDATE group_association
    SET joined = current_timestamp at time zone 'utc', hasleft = false
    WHERE userid = v_userid AND conversationid = v_conversationId
    RETURNING colorIndex INTO v_res_colorIndex;
  ELSE
    select colorIndex 
    from group_association g
    join (
      select conversationid, max(joined) as lastjoined
      from group_association
      group by conversationid
    ) as tmp ON g.conversationid = tmp.conversationid AND g.joined = tmp.lastjoined
    where g.conversationid = v_conversationId
    INTO v_colorIndex;

    UPDATE group_association
    SET joined = current_timestamp at time zone 'utc', colorindex = v_colorIndex + 1, hasleft = false
    WHERE userid = v_userid AND conversationid = v_conversationId
    RETURNING colorIndex INTO v_res_colorIndex;
  END IF;

  RETURN v_res_colorIndex;
end;
$$ language PLpgSQL;



create or replace procedure deleteAccount(
    in v_userid integer)
AS $$
DECLARE v_newAdmin_userid group_association.userid%TYPE;
DECLARE v_newAdmin_conversationid group_association.conversationId%TYPE;
DECLARE c RECORD;
begin
    
  UPDATE public.user 
  SET isdeleted = true, email = '', password = '', recovery_uuid = null
  WHERE id = v_userid;

  FOR c IN (
    select a.conversationid as id, count(*) as numberOfAdmins
    FROM (
      SELECT *
      FROM public.group_association
      where conversationid in (
        -- Hole die Zeilen in denen ich Mitglied bin als auch admin
        select conversationid
        from public.group_association
        where userid = v_userid AND isadmin = true
      )
    ) a
    WHERE a.isadmin = true
    GROUP BY a.conversationid
  )
  LOOP
    IF c.numberOfAdmins = 1 THEN
      
      select userid, conversationid INTO v_newAdmin_userid, v_newAdmin_conversationid
      from public.group_association
      where userid != v_userid and conversationid = c.id
      order by joined asc
      limit 1;

      UPDATE public.group_association
      SET isadmin = true
      WHERE conversationId = v_newAdmin_conversationid and userid = v_newAdmin_userid;

    END IF;
  END LOOP;

  UPDATE public.group_association
  SET isadmin = false, hasleft = true
  WHERE userid = v_userid AND isadmin = true;
end;
$$ language PLpgSQL;


create or replace function createConversation(
    in v_userid integer,
    in v_title varchar(100),
    in v_repourl text,
    in v_initialmembers integer[])
RETURNS integer
AS $$
DECLARE member integer;
DECLARE v_new_conversationid integer;
begin
  	INSERT INTO conversation (title, repourl) 
		VALUES (v_title, v_repourl) 
		RETURNING id INTO v_new_conversationid;

		INSERT INTO group_association (isadmin, userid, conversationid, joined, colorIndex) 
		VALUES (true, v_userid, v_new_conversationid, current_timestamp at time zone 'utc', 0);

		FOREACH member IN ARRAY v_initialmembers
    LOOP
				INSERT INTO group_association (isadmin, userid, conversationid, joined, colorIndex) 
				VALUES (false, member, v_new_conversationid, NULL, -1);
		END LOOP;

    RETURN v_new_conversationid;
end;
$$ language PLpgSQL;


CREATE or replace FUNCTION public."updateLastOnline"()
    RETURNS trigger
    LANGUAGE 'plpgsql'
     NOT LEAKPROOF
AS $BODY$
BEGIN
update public.user
set lastonline = current_timestamp at time zone 'utc'
where id = new.id;
RETURN NULL;
END;
$BODY$;

CREATE  or replace FUNCTION public.calculateUnreadMessages(IN v_conversationid integer, IN v_userid integer)
    RETURNS integer
    LANGUAGE 'plpgsql'
AS $BODY$
BEGIN
SELECT count(*) FROM message_status
WHERE message_status.hasread = false and 
	userid = v_userid and 
	conversationid = v_conversationid;
END
	$BODY$;


CREATE TRIGGER "afterSetOnline"
AFTER UPDATE OF isonline
ON public."user"
FOR EACH ROW
WHEN (NEW.isonline = true)
EXECUTE PROCEDURE public."updateLastOnline"();


create or replace procedure inviteUser(
    in v_userid integer, in v_conversationId integer)
AS $$
DECLARE v_exists Boolean;
begin
  select exists(
    select 1 from group_association 
    where conversationId = v_conversationId and userid = v_userid
  ) into v_exists

  if v_exists = true THEN
    UPDATE group_association
		SET hasleft = false, joined = null
		WHERE userid = v_userid AND conversationid = v_conversationId;
  else
    INSERT INTO group_association (isadmin, userid, conversationid, joined, colorindex) 
		VALUES (false, v_userid, v_conversationId, NULL, -1);
  end if;
end;
$$ language PLpgSQL;
