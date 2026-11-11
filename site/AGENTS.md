# Agent Guidelines

## Astro Development

When starting the dev server, use background mode:

```
astro dev --background
```

Manage the background server with `astro dev stop`, `astro dev status`, and `astro dev logs`.

## Book Discipline

1. Use **zh-Hants** to write articles.
2. Trigger skill **write-article** when user ask you to write and edit articles in book.
3. The posts will put in `site/src/contents/docs/<fields>/<topic>/<chapter>`, and code example will put in `playground/<language>/<topic>/<chapter>`
4. Please read the table of contents in each field first before writing.