create or replace function insert_comment(
	p_user_id bigint,
	p_post_id bigint,
	p_parent_path ltree,
	p_body text,
	p_upvotes bigint default 0,
	p_downvotes bigint default 0
) returns comments as $$
declare
	new_id bigint;
	new_path ltree;
	new_depth int;
	result comments;
begin
	new_id := nextval('comments_id_seq');
	if p_parent_path is null then
		new_path := new_id::text::ltree;
		new_depth := 0;
	else
		new_path := (p_parent_path::text || '.' || new_id::text)::ltree;
    	new_depth := nlevel(p_parent_path);
	end if;

	insert into comments (id, user_id, post_id, path, depth, body, upvotes, downvotes)
	values (new_id, p_user_id, p_post_id, new_path, new_depth, p_body, p_upvotes, p_downvotes)
	returning * into result;

	return result;
end;

$$language plpgsql;