package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	db *sql.DB
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// GetPaidGroupsWithCredentials mendapatkan group yang sudah lunas beserta kredensial
func (h *SubscriptionHandler) GetPaidGroupsWithCredentials(c *gin.Context) {
	// User ID is already checked by middleware

	// Get query parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")

	pageInt := 1
	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageInt = p
	}
	pageSizeInt := 20
	if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
		pageSizeInt = ps
	}

	offset := (pageInt - 1) * pageSizeInt

	// Build query for paid groups
	query := `
		SELECT 
			g.id, g.name, g.description, g.app_id, g.max_members, 
			g.price_per_member, g.total_price, g.group_status, g.all_paid_at, g.created_at,
			a.id, a.name, a.icon_url
		FROM groups g
		JOIN apps a ON g.app_id = a.id
		WHERE g.group_status = 'paid_group'
	`
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		query += fmt.Sprintf(" AND (g.name ILIKE $%d OR a.name ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY g.all_paid_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSizeInt, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch paid groups"})
		return
	}
	defer rows.Close()

	var groups []models.PaidGroupWithCredentials
	for rows.Next() {
		var group models.PaidGroupWithCredentials
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.AppID,
			&group.MaxMembers,
			&group.PricePerMember,
			&group.TotalPrice,
			&group.GroupStatus,
			&group.AllPaidAt,
			&group.CreatedAt,
			&group.App.ID,
			&group.App.Name,
			&group.App.IconURL,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan group"})
			return
		}

		// Get members for this group
		members, err := h.getGroupMembers(group.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
			return
		}
		group.Members = members
		group.MemberCount = len(members)

		// Get account credentials for this group
		credentials, err := h.getGroupCredentials(group.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group credentials"})
			return
		}
		group.AccountCredentials = credentials

		groups = append(groups, group)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM groups g
		JOIN apps a ON g.app_id = a.id
		WHERE g.group_status = 'paid_group'
	`
	countArgs := []interface{}{}
	countArgIndex := 1

	if search != "" {
		countQuery += fmt.Sprintf(" AND (g.name ILIKE $%d OR a.name ILIKE $%d)", countArgIndex, countArgIndex)
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count groups"})
		return
	}

	response := models.GetPaidGroupsResponse{
		Groups:   groups,
		Total:    total,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Paid groups retrieved successfully",
		"data":    response,
	})
}

// getGroupMembers mendapatkan members untuk group tertentu
func (h *SubscriptionHandler) getGroupMembers(groupID string) ([]models.GroupMemberWithUser, error) {
	query := `
		SELECT 
			gm.id, gm.group_id, gm.user_id, gm.user_status, gm.joined_at,
			u.id, u.full_name, u.email, u.avatar_url
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1
		ORDER BY gm.joined_at ASC
	`

	rows, err := h.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.GroupMemberWithUser
	for rows.Next() {
		var member models.GroupMemberWithUser
		err := rows.Scan(
			&member.ID,
			&member.GroupID,
			&member.UserID,
			&member.UserStatus,
			&member.JoinedAt,
			&member.User.ID,
			&member.User.FullName,
			&member.User.Email,
			&member.User.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// getGroupCredentials mendapatkan account credentials untuk group tertentu
func (h *SubscriptionHandler) getGroupCredentials(groupID string) ([]models.AccountCredentialWithUser, error) {
	query := `
		SELECT 
			ac.id, ac.group_id, ac.user_id, ac.username, ac.email, ac.description, ac.created_at,
			u.id, u.full_name, u.email, u.avatar_url
		FROM account_credentials ac
		JOIN users u ON ac.user_id = u.id
		WHERE ac.group_id = $1
		ORDER BY ac.created_at DESC
	`

	rows, err := h.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []models.AccountCredentialWithUser
	for rows.Next() {
		var credential models.AccountCredentialWithUser
		err := rows.Scan(
			&credential.ID,
			&credential.GroupID,
			&credential.UserID,
			&credential.Username,
			&credential.Email,
			&credential.Description,
			&credential.CreatedAt,
			&credential.User.ID,
			&credential.User.FullName,
			&credential.User.Email,
			&credential.User.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}

	return credentials, nil
}
