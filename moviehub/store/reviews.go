package store

import (
    "context"
    "errors"
    "time"

    "github.com/jackc/pgx/v5/pgconn"
)
type Review struct {
    ID        int64     `json:"id"`
    MemberID  int64     `json:"member_id"`
    MovieID   int64     `json:"movie_id"`
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

var (
    ErrDuplicate = errors.New("duplicate")
    ErrNoMovie   = errors.New("movie not found")
)

type ReviewAuthor struct {
    ID          int64     `json:"id"`
    DisplayName string    `json:"display_name"`
    AvatarURL   *string   `json:"avatar_url"`
}

type ReviewItem struct {
    Review
    Author ReviewAuthor `json:"author"`
}

func (s *Store) ListReviews(ctx context.Context, movieID int64) ([]ReviewItem, error) {
    rows, err := s.Pool.Query(ctx, `
        SELECT
            r.id,
            r.member_id,
            r.movie_id,
            r.text,
            r.created_at,
            r.updated_at,
            m.id,
            m.display_name,
            m.avatar_url
        FROM reviews r
        JOIN members m ON m.id = r.member_id
        WHERE r.movie_id = $1
        ORDER BY r.created_at DESC
    `, movieID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    items := make([]ReviewItem, 0)
    for rows.Next() {
        var item ReviewItem

        err := rows.Scan(
            &item.ID,
            &item.MemberID,
            &item.MovieID,
            &item.Text,
            &item.CreatedAt,
            &item.UpdatedAt,
            &item.Author.ID,
            &item.Author.DisplayName,
            &item.Author.AvatarURL,
        )
        if err != nil {
            return nil, err
        }

        items = append(items, item)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return items, nil
}

func (s *Store) AddReview(ctx context.Context, memberID, movieID int64, text string) (Review, error) {
    r := Review{MemberID: memberID, MovieID: movieID, Text: text}
    err := s.Pool.QueryRow(ctx,
        `INSERT INTO reviews (member_id, movie_id, text)
         VALUES ($1, $2, $3)
         RETURNING id, created_at, updated_at`, memberID, movieID, text).
        Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // ผิดกติกาห้ามซ้ำ
            return Review{}, ErrDuplicate
        case "23503": // ผิดกติกาคีย์นอก หนังไม่มีจริง
            return Review{}, ErrNoMovie
        }
    }
    return r, err
}