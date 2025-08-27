#!/bin/bash

src_files=(
  "schema"
  "indexes"
  "insert_comment"
  "get_comments_by_popularity"
  "vote_comment"
  "vote_post"
  "delete_comment_vote"
  "delete_post_vote"
)

> doc/init.sql

for file in "${src_files[@]}"; do 
  cat "doc/$file.sql" >> "doc/init.sql"
done