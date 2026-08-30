package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrMemberExists = errors.New("member already exists")
)

type store interface {
	CreateGroupWithAdmin(ctx context.Context, g *Group, adminUserID uuid.UUID) (*Group, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
	FindLogo(ctx context.Context, id uuid.UUID) (*Logo, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Group, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]*Member, error)
	FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error)
	FindMembershipByEmail(ctx context.Context, groupID uuid.UUID, email string) (*Membership, error)
	AddMember(ctx context.Context, groupID, userID uuid.UUID, role string) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	Update(ctx context.Context, g *Group) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateInvitation(ctx context.Context, inv *Invitation) error
	CreateInvitations(ctx context.Context, invites []*Invitation) ([]*Invitation, error)
	MembersByEmails(ctx context.Context, groupID uuid.UUID, emails []string) (map[string]bool, error)
	PendingInvitationsByEmails(ctx context.Context, groupID uuid.UUID, emails []string) (map[string]bool, error)
	FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error)
	FindPendingInvitation(ctx context.Context, groupID uuid.UUID, email string) (*Invitation, error)
	AcceptInvitation(ctx context.Context, inv *Invitation, userID uuid.UUID) error
	FindActiveInviteLink(ctx context.Context, groupID uuid.UUID) (*InviteLink, error)
	CreateInviteLink(ctx context.Context, link *InviteLink) error
	RevokeInviteLinks(ctx context.Context, groupID uuid.UUID) error
	FindInviteLinkByTokenHash(ctx context.Context, tokenHash string) (*InviteLink, error)
	FindGroupPreview(ctx context.Context, groupID uuid.UUID) (*GroupPreview, error)
	JoinViaInviteLink(ctx context.Context, link *InviteLink, userID uuid.UUID) error
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const groupColumns = "id::text, name, description, currency, created_by::text, created_at, updated_at, logo_image IS NOT NULL"

func scanGroup(row pgx.Row) (*Group, error) {
	var (
		g     Group
		rawID string
	)

	if err := row.Scan(&rawID, &g.Name, &g.Description, &g.Currency, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.HasLogo); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	g.ID = id

	return &g, nil
}

func (r *Repository) CreateGroupWithAdmin(ctx context.Context, g *Group, adminUserID uuid.UUID) (*Group, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO groups (name, description, currency, created_by, logo_image, logo_content_type)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''::bytea), NULLIF($6, ''))
		 RETURNING `+groupColumns,
		g.Name, g.Description, g.Currency, adminUserID.String(), g.LogoImage, g.LogoContentType)

	created, err := scanGroup(row)
	if err != nil {
		return nil, fmt.Errorf("insert group: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)`,
		created.ID.String(), adminUserID.String(), RoleAdmin); err != nil {
		return nil, fmt.Errorf("insert admin member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Group, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+groupColumns+`,
		        (SELECT count(*) FROM group_members gm WHERE gm.group_id = g.id)
		 FROM groups g
		 WHERE g.id = $1`,
		id.String())

	var (
		g           Group
		rawID       string
		memberCount int
	)

	if err := row.Scan(&rawID, &g.Name, &g.Description, &g.Currency, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.HasLogo, &memberCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find group by id: %w", err)
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	g.ID = parsedID
	g.MemberCount = memberCount

	return &g, nil
}

func (r *Repository) FindLogo(ctx context.Context, id uuid.UUID) (*Logo, error) {
	var (
		contentType *string
		image       []byte
	)

	err := r.pool.QueryRow(ctx,
		`SELECT NULLIF(logo_content_type, ''), logo_image
		 FROM groups
		 WHERE id = $1`,
		id.String()).Scan(&contentType, &image)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find group logo: %w", err)
	}
	if contentType == nil {
		return nil, ErrNoLogo
	}

	return &Logo{Image: image, ContentType: *contentType}, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id::text, g.name, g.description, g.currency, g.created_by::text, g.created_at, g.updated_at, g.logo_image IS NOT NULL, gm.role,
		        (SELECT count(*) FROM group_members m WHERE m.group_id = g.id)
		 FROM groups g
		 JOIN group_members gm ON gm.group_id = g.id
		 WHERE gm.user_id = $1
		 ORDER BY g.created_at DESC`,
		userID.String())
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	groups := []*Group{}
	for rows.Next() {
		var (
			g     Group
			rawID string
		)

		if err := rows.Scan(&rawID, &g.Name, &g.Description, &g.Currency, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.HasLogo, &g.Role, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("parse group id: %w", err)
		}
		g.ID = id
		groups = append(groups, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return groups, nil
}

func (r *Repository) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*Member, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id::text, u.name, u.email, gm.role, gm.joined_at
		 FROM group_members gm
		 JOIN users u ON u.id = gm.user_id
		 WHERE gm.group_id = $1
		 ORDER BY gm.joined_at`,
		groupID.String())
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	members := []*Member{}
	for rows.Next() {
		var (
			m     Member
			rawID string
		)

		if err := rows.Scan(&rawID, &m.Name, &m.Email, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("parse member id: %w", err)
		}
		m.UserID = id
		members = append(members, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return members, nil
}

func (r *Repository) FindMembership(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT group_id::text, user_id::text, role
		 FROM group_members
		 WHERE group_id = $1 AND user_id = $2`,
		groupID.String(), userID.String())

	var (
		m        Membership
		rawGroup string
		rawUser  string
	)

	if err := row.Scan(&rawGroup, &rawUser, &m.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find membership: %w", err)
	}

	groupIDParsed, err := uuid.Parse(rawGroup)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	userIDParsed, err := uuid.Parse(rawUser)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	m.GroupID = groupIDParsed
	m.UserID = userIDParsed

	return &m, nil
}

func (r *Repository) FindMembershipByEmail(ctx context.Context, groupID uuid.UUID, email string) (*Membership, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT gm.group_id::text, gm.user_id::text, gm.role
		 FROM group_members gm
		 JOIN users u ON u.id = gm.user_id
		 WHERE gm.group_id = $1 AND u.email = $2`,
		groupID.String(), email)

	var (
		m        Membership
		rawGroup string
		rawUser  string
	)

	if err := row.Scan(&rawGroup, &rawUser, &m.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find membership by email: %w", err)
	}

	groupIDParsed, err := uuid.Parse(rawGroup)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	userIDParsed, err := uuid.Parse(rawUser)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	m.GroupID = groupIDParsed
	m.UserID = userIDParsed

	return &m, nil
}

func (r *Repository) AddMember(ctx context.Context, groupID, userID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)`,
		groupID.String(), userID.String(), role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrMemberExists
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`,
		groupID.String(), userID.String())
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, g *Group) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE groups SET name = $1, description = $2, currency = $3, updated_at = now()
		 WHERE id = $4`,
		g.Name, g.Description, g.Currency, g.ID.String())
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id.String())
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateInvitation(ctx context.Context, inv *Invitation) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO group_invitations (group_id, email, invited_by, token_hash, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id::text, group_id::text, email, invited_by::text, token_hash, status, expires_at, created_at`,
		inv.GroupID.String(), inv.Email, inv.InvitedBy.String(), inv.TokenHash, inv.Status, inv.ExpiresAt)

	var (
		rawID      string
		rawGroupID string
		rawInvited string
	)

	if err := row.Scan(&rawID, &rawGroupID, &inv.Email, &rawInvited, &inv.TokenHash, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
		return fmt.Errorf("insert invitation: %w", err)
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse invitation id: %w", err)
	}
	groupID, err := uuid.Parse(rawGroupID)
	if err != nil {
		return fmt.Errorf("parse invitation group id: %w", err)
	}
	invitedBy, err := uuid.Parse(rawInvited)
	if err != nil {
		return fmt.Errorf("parse invited by: %w", err)
	}
	inv.ID = id
	inv.GroupID = groupID
	inv.InvitedBy = invitedBy

	return nil
}

func (r *Repository) CreateInvitations(ctx context.Context, invites []*Invitation) ([]*Invitation, error) {
	if len(invites) == 0 {
		return nil, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	created := make([]*Invitation, 0, len(invites))
	for _, inv := range invites {
		row := tx.QueryRow(ctx,
			`INSERT INTO group_invitations (group_id, email, invited_by, token_hash, status, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, group_id::text, email, invited_by::text, token_hash, status, expires_at, created_at`,
			inv.GroupID.String(), inv.Email, inv.InvitedBy.String(), inv.TokenHash, inv.Status, inv.ExpiresAt)

		var (
			rawID      string
			rawGroupID string
			rawInvited string
		)
		if err := row.Scan(&rawID, &rawGroupID, &inv.Email, &rawInvited, &inv.TokenHash, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("insert invitation: %w", err)
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("parse invitation id: %w", err)
		}
		groupID, err := uuid.Parse(rawGroupID)
		if err != nil {
			return nil, fmt.Errorf("parse invitation group id: %w", err)
		}
		invitedBy, err := uuid.Parse(rawInvited)
		if err != nil {
			return nil, fmt.Errorf("parse invited by: %w", err)
		}
		inv.ID = id
		inv.GroupID = groupID
		inv.InvitedBy = invitedBy
		created = append(created, inv)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (r *Repository) MembersByEmails(ctx context.Context, groupID uuid.UUID, emails []string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.email
		 FROM group_members gm
		 JOIN users u ON u.id = gm.user_id
		 WHERE gm.group_id = $1 AND u.email = ANY($2)`,
		groupID.String(), emails)
	if err != nil {
		return nil, fmt.Errorf("query members by emails: %w", err)
	}
	defer rows.Close()

	members := make(map[string]bool, len(emails))
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("scan member email: %w", err)
		}
		members[email] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members by emails: %w", err)
	}

	return members, nil
}

func (r *Repository) PendingInvitationsByEmails(ctx context.Context, groupID uuid.UUID, emails []string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT email
		 FROM group_invitations
		 WHERE group_id = $1 AND email = ANY($2) AND status = 'pending'`,
		groupID.String(), emails)
	if err != nil {
		return nil, fmt.Errorf("query pending invitations by emails: %w", err)
	}
	defer rows.Close()

	pending := make(map[string]bool, len(emails))
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("scan invitation email: %w", err)
		}
		pending[email] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending invitations by emails: %w", err)
	}

	return pending, nil
}

func (r *Repository) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id::text, group_id::text, email, invited_by::text, token_hash, status, expires_at, created_at
		 FROM group_invitations
		 WHERE token_hash = $1`,
		tokenHash)

	inv, err := scanInvitation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invitation by token hash: %w", err)
	}

	return inv, nil
}

func (r *Repository) FindPendingInvitation(ctx context.Context, groupID uuid.UUID, email string) (*Invitation, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id::text, group_id::text, email, invited_by::text, token_hash, status, expires_at, created_at
		 FROM group_invitations
		 WHERE group_id = $1 AND email = $2 AND status = 'pending'
		 ORDER BY created_at DESC
		 LIMIT 1`,
		groupID.String(), email)

	inv, err := scanInvitation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find pending invitation: %w", err)
	}

	return inv, nil
}

func scanInvitation(row pgx.Row) (*Invitation, error) {
	var (
		inv        Invitation
		rawID      string
		rawGroupID string
		rawInvited string
	)

	if err := row.Scan(&rawID, &rawGroupID, &inv.Email, &rawInvited, &inv.TokenHash, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse invitation id: %w", err)
	}
	groupID, err := uuid.Parse(rawGroupID)
	if err != nil {
		return nil, fmt.Errorf("parse invitation group id: %w", err)
	}
	invitedBy, err := uuid.Parse(rawInvited)
	if err != nil {
		return nil, fmt.Errorf("parse invited by: %w", err)
	}
	inv.ID = id
	inv.GroupID = groupID
	inv.InvitedBy = invitedBy

	return &inv, nil
}

func (r *Repository) AcceptInvitation(ctx context.Context, inv *Invitation, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		inv.GroupID.String(), userID.String(), RoleMember)
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMemberExists
	}

	if _, err := tx.Exec(ctx,
		`UPDATE group_invitations SET status = 'accepted' WHERE id = $1`,
		inv.ID.String()); err != nil {
		return fmt.Errorf("update invitation status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

const inviteLinkColumns = "id::text, group_id::text, token, token_hash, created_by::text, expires_at, revoked_at, max_uses, used_count, created_at"

func scanInviteLink(row pgx.Row) (*InviteLink, error) {
	var (
		link       InviteLink
		rawID      string
		rawGroupID string
		rawCreator string
	)

	if err := row.Scan(&rawID, &rawGroupID, &link.Token, &link.TokenHash, &rawCreator, &link.ExpiresAt, &link.RevokedAt, &link.MaxUses, &link.UsedCount, &link.CreatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse invite link id: %w", err)
	}
	groupID, err := uuid.Parse(rawGroupID)
	if err != nil {
		return nil, fmt.Errorf("parse invite link group id: %w", err)
	}
	createdBy, err := uuid.Parse(rawCreator)
	if err != nil {
		return nil, fmt.Errorf("parse invite link creator id: %w", err)
	}
	link.ID = id
	link.GroupID = groupID
	link.CreatedBy = createdBy

	return &link, nil
}

func (r *Repository) FindActiveInviteLink(ctx context.Context, groupID uuid.UUID) (*InviteLink, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+inviteLinkColumns+`
		 FROM group_invite_links
		 WHERE group_id = $1 AND revoked_at IS NULL AND expires_at > now()`,
		groupID.String())

	link, err := scanInviteLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find active invite link: %w", err)
	}

	return link, nil
}

func (r *Repository) CreateInviteLink(ctx context.Context, link *InviteLink) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE group_invite_links SET revoked_at = now()
		 WHERE group_id = $1 AND revoked_at IS NULL`,
		link.GroupID.String()); err != nil {
		return fmt.Errorf("revoke previous invite links: %w", err)
	}

	row := tx.QueryRow(ctx,
		`INSERT INTO group_invite_links (group_id, token, token_hash, created_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+inviteLinkColumns,
		link.GroupID.String(), link.Token, link.TokenHash, link.CreatedBy.String(), link.ExpiresAt)

	created, err := scanInviteLink(row)
	if err != nil {
		return fmt.Errorf("insert invite link: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	*link = *created

	return nil
}

func (r *Repository) RevokeInviteLinks(ctx context.Context, groupID uuid.UUID) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE group_invite_links SET revoked_at = now()
		 WHERE group_id = $1 AND revoked_at IS NULL`,
		groupID.String()); err != nil {
		return fmt.Errorf("revoke invite links: %w", err)
	}
	return nil
}

func (r *Repository) FindInviteLinkByTokenHash(ctx context.Context, tokenHash string) (*InviteLink, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+inviteLinkColumns+`
		 FROM group_invite_links
		 WHERE token_hash = $1`,
		tokenHash)

	link, err := scanInviteLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invite link by token hash: %w", err)
	}

	return link, nil
}

func (r *Repository) FindGroupPreview(ctx context.Context, groupID uuid.UUID) (*GroupPreview, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT g.id::text, g.name, g.description, g.currency,
		        (SELECT count(*) FROM group_members gm WHERE gm.group_id = g.id),
		        creator.name,
		        (SELECT COALESCE(array_agg(u2.name ORDER BY gm2.joined_at), '{}'::text[])
		         FROM group_members gm2 JOIN users u2 ON u2.id = gm2.user_id
		         WHERE gm2.group_id = g.id),
		        g.logo_image IS NOT NULL,
		        g.created_at
		 FROM groups g
		 JOIN users creator ON creator.id = g.created_by
		 WHERE g.id = $1`,
		groupID.String())

	var (
		preview    GroupPreview
		rawGroupID string
	)
	if err := row.Scan(&rawGroupID, &preview.Name, &preview.Description, &preview.Currency, &preview.MemberCount, &preview.CreatorName, &preview.MemberNames, &preview.HasLogo, &preview.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find group preview: %w", err)
	}

	id, err := uuid.Parse(rawGroupID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}
	preview.GroupID = id

	return &preview, nil
}

func (r *Repository) JoinViaInviteLink(ctx context.Context, link *InviteLink, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		link.GroupID.String(), userID.String(), RoleMember)
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMemberExists
	}

	ct, err = tx.Exec(ctx,
		`UPDATE group_invite_links SET used_count = used_count + 1
		 WHERE id = $1 AND revoked_at IS NULL AND expires_at > now() AND (max_uses IS NULL OR used_count < max_uses)`,
		link.ID.String())
	if err != nil {
		return fmt.Errorf("update invite link usage: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrInviteLinkLimit
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
