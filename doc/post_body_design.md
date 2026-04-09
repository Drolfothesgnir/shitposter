# Post Body Design

This note captures the current decisions around post structure, inline markup, storage, rendering, and editing.

## Goals

- Support block-based posts.
- Use `scum` for inline markup inside text-bearing blocks.
- Keep one canonical source format for create, edit, and persistence.
- Allow fast public reads with rendered HTML.
- Support a WYSIWYG editor with live preview.
- Keep backend and frontend parsing behavior consistent.

## Separation of concerns

- `shit` owns the block document format.
- `scum` owns inline markup parsing.
- `SerializableNode` is parser output, not render output.
- Rendering is a separate concern from parsing and storage.

In other words:

- block structure answers "what kinds of content does the post contain?"
- `scum` answers "how is inline text marked up?"
- renderers answer "how should this be shown?"

## Canonical source format

The canonical source of truth should be the post body JSON document.

- Store the post body as JSONB in `posts.body`.
- For text-bearing blocks, store raw `scum` source strings.
- Do not make rendered HTML the source of truth.
- Do not make serialized parser trees the source of truth.

Recommended shape:

```json
{
  "version": 1,
  "blocks": [
    {
      "id": "b1",
      "type": "paragraph",
      "content": "This is my $_first_$ post!"
    },
    {
      "id": "b2",
      "type": "image",
      "src": "cat.png",
      "alt": "cat",
      "caption": ""
    }
  ]
}
```

Notes:

- `version` is worth having from day one.
- `id` is useful for editor stability, drag/drop, selection restore, and React keys.
- `type` should be a readable string like `"paragraph"` or `"image"`, not `0`, `1`, etc.
- For uploaded media, a future `asset_id` is usually better than raw URL strings.

## Why dedicated block structs

Dedicated block structs are preferred over one loose block object with many optional fields.

Good direction:

- `ParagraphBlock`
- `ImageBlock`
- `QuoteBlock`
- `CodeBlock`

Advantages:

- clearer validation
- fewer invalid field combinations
- easier API evolution
- easier frontend rendering

## Example create payload

If a user creates a post with:

- title: `my first post`
- block 1: paragraph `This is my $_first_$ post!`
- block 2: image `cat.png`

then `POST /posts` should receive something like:

```json
{
  "title": "my first post",
  "topics": [],
  "body": {
    "version": 1,
    "blocks": [
      {
        "id": "b1",
        "type": "paragraph",
        "content": "This is my $_first_$ post!"
      },
      {
        "id": "b2",
        "type": "image",
        "src": "cat.png",
        "alt": "cat",
        "caption": ""
      }
    ]
  }
}
```

## Write path

When the backend receives a post create or update request:

1. Validate the block document shape.
2. Validate each block by type.
3. For each text-bearing block, parse the raw `scum` source.
4. Reject invalid or unsafe input, or store warnings if the chosen UX allows soft warnings.
5. Render sanitized HTML if a cached read representation is desired.
6. Persist the canonical body JSON and any derived cached fields.

Important:

- parse and validate before persistence
- sanitize rendered HTML before returning or storing it
- keep the backend authoritative even if the frontend also parses

## What to store in the database

Recommended baseline:

- `posts.body JSONB NOT NULL` as canonical source
- `posts.body_html TEXT NOT NULL` as cached rendered HTML
- `posts.render_version INT NOT NULL` to invalidate stale renders when rules change
- optionally `posts.excerpt TEXT` for feeds and previews

Why a column for rendered HTML is fine:

- it is simple
- it keeps the first version easy to ship
- it avoids introducing Redis or a second cache system too early

Treat `body_html` as a derived cache, not as truth.

If the project later needs multiple render variants, a dedicated `post_renders` table can be introduced.

## Public read vs edit vs preview

Use different representations for different paths.

### Public read API

Public reads should prefer rendered output.

- return sanitized HTML for the full post page
- return excerpts or summaries for feed/list pages
- avoid returning full canonical bodies in list pages if payload size matters

If there is a strict response-size target like `14 KB`, the feed endpoints should return excerpts rather than full post bodies.

### Edit API

When a user edits an existing post, return the canonical source JSON.

Example:

```json
{
  "id": 42,
  "title": "my first post",
  "topics": [],
  "body": {
    "version": 1,
    "blocks": [
      {
        "id": "b1",
        "type": "paragraph",
        "content": "This is my $_first_$ post!"
      },
      {
        "id": "b2",
        "type": "image",
        "src": "cat.png",
        "alt": "cat",
        "caption": ""
      }
    ]
  }
}
```

Why:

- this is the authoring format
- it preserves user intent
- it avoids reconstructing source from HTML
- it keeps the server contract stable

### Preview API

Preview is derived output from draft source.

Possible preview response:

- rendered HTML
- parse warnings
- optional parser tree if the editor needs it

Preview should not replace canonical storage.

## WYSIWYG editor decision

A WYSIWYG editor does not change the server-side source of truth.

Recommended model:

- frontend editor uses whatever rich local state is convenient
- save converts editor state back to canonical JSON
- backend stores canonical JSON
- public reads use rendered HTML

So even with toolbar buttons like `B`, `i`, underline, link, image, and quote:

- the server should still receive canonical JSON
- text blocks should still end up as raw `scum` source in the persisted body

## Frontend preview and parser reuse

For live preview, the frontend needs local parsing or an equivalent editor-state render path.

Because `scum` already exists in Go and reimplementing it in TypeScript is undesirable, WebAssembly is a reasonable choice.

Reason to use WASM:

- exact parser consistency between backend and frontend
- same warnings and edge-case behavior
- no second parser implementation to maintain

Reason not to choose WASM:

- not because it is always faster
- bundle size and initialization cost still matter

Recommended approach:

- lazy-load `scum.wasm` only on create/edit pages
- keep the backend authoritative
- parse only changed text blocks when possible
- debounce preview updates slightly
- move parsing into a Web Worker if typing latency becomes noticeable

Summary:

- use WASM for consistency and reuse
- do not rely on it as the sole validator
- server still reparses on save

## Serializable tree guidance

`SerializableNode` is useful as a transport or tooling shape, but it should not be the canonical persisted post body.

Good uses:

- editor helper data
- preview/debug output
- frontend tooling

Not recommended as the persisted source of truth:

- parser output is still derived from the raw authoring input
- authoring should remain editable without depending on parser internals

## Current serializer semantics

Current design intent for serialized parser output:

- the tree describes parser structure, not render semantics
- `Name` identifies the semantic node name
- `ID` is the numeric trigger byte
- `Content` is parser payload/data, not guaranteed final display text
- renderers decide how to interpret a node

This is important for greedy tags and similar cases:

- a parser node may carry content that is meaningful only to a specific renderer
- the parser should not overreach into presentation logic

## Practical recommendations

- Keep `posts.body` as canonical block JSON.
- Keep raw `scum` source inside text blocks.
- Cache rendered sanitized HTML in a DB column.
- Return canonical JSON on edit paths.
- Return rendered HTML or excerpts on read paths.
- Use WASM on editor pages if you want exact parser parity.
- Keep backend validation and rendering authoritative.

## Open implementation questions

These are still implementation details to settle later:

- exact Go structs for block types in `shit`
- whether images are addressed by `src`, `asset_id`, or both
- whether warnings are stored or returned only during preview/save
- exact preview response shape
- whether excerpts are stored eagerly or computed asynchronously
