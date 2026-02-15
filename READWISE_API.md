# Readwise Reader API Documentation (V3)

This document summarizes the Readwise Reader API based on the official documentation at `https://readwise.io/reader_api`.

## Authentication

All requests require an `Authorization` header with your access token:

```http
Authorization: Token <YOUR_ACCESS_TOKEN>
```

You can get your token from `readwise.io/access_token`.

To verify your token:
- **Request**: `GET https://readwise.io/api/v2/auth/`
- **Response**: `204 No Content` (success)

---

## Document CREATE

Save a new document to Reader.

- **Endpoint**: `POST https://readwise.io/api/v3/save/`
- **Payload (JSON)**:
    - `url` (string, **required**): The document's unique URL.
    - `html` (string, optional): Valid HTML content.
    - `should_clean_html` (boolean, optional): Default `false`.
    - `title` (string, optional): Overwrite document title.
    - `author` (string, optional): Overwrite author.
    - `summary` (string, optional): Summary text.
    - `published_date` (ISO 8601 string, optional): e.g., `"2020-07-14T20:11:24+00:00"`.
    - `image_url` (string, optional): Cover image URL.
    - `location` (string, optional): One of `new`, `later`, `archive`, `feed`. Default is `new`.
    - `category` (string, optional): One of `article`, `email`, `rss`, `highlight`, `note`, `pdf`, `epub`, `tweet`, `video`.
    - `tags` (list of strings, optional): e.g., `["tag1", "tag2"]`.
    - `notes` (string, optional): Top-level note.

---

## Document LIST

Fetch your documents.

- **Endpoint**: `GET https://readwise.io/api/v3/list/`
- **Query Parameters**:
    - `id` (string): Return just one document by ID.
    - `updatedAfter` (ISO 8601 string): documents updated after this date.
    - `location` (string): One of `new`, `later`, `shortlist`, `archive`, `feed`.
    - `category` (string): One of `article`, `email`, `rss`, etc.
    - `tag` (string): Filter by tag.
    - `limit` (integer): 1-100 (Default 100).
    - `pageCursor` (string): For pagination.
    - `withHtmlContent` (boolean): Include HTML content in response.

---

## Document UPDATE

Update specific fields of an existing document.

- **Endpoint**: `PATCH https://readwise.io/api/v3/update/<document_id>/`
- **Payload (JSON)**:
    - `title` (string)
    - `author` (string)
    - `summary` (string)
    - `published_date` (date string)
    - `image_url` (string)
    - `seen` (boolean): Mark as seen/unseen.
    - `location` (string): One of `new`, `later`, `archive`, `feed`.
    - `category` (string)
    - `tags` (list of strings)
    - `notes` (string)

---

## Document DELETE

- **Endpoint**: `DELETE https://readwise.io/api/v3/delete/<document_id>/`
- **Response**: `204 No Content`

---

## Rate Limiting

- **Base Rate**: 20 requests per minute.
- **CREATE/UPDATE Endpoints**: 50 requests per minute.
- **Note**: Check the `Retry-After` header in `429` responses for the number of seconds to wait.

---

## Webhooks

Reader supports webhooks for real-time notifications on highlights and documents. Configure them in your account settings.
