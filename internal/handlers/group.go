package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"salome-be/internal/models"
	"salome-be/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupHandler struct {
	db              *sql.DB
	stateMachineSvc *services.StateMachineService
}

func NewGroupHandler(db *sql.DB) *GroupHandler {
	return &GroupHandler{
		db:              db,
		stateMachineSvc: services.NewStateMachineService(db),
	}
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get app information to calculate pricing
	var app models.App
	appQuery := `
		SELECT id, name, total_price, max_group_members
		FROM apps WHERE id = $1 AND is_active = true
	`
	err := h.db.QueryRow(appQuery, req.AppID).Scan(
		&app.ID, &app.Name, &app.TotalPrice, &app.MaxGroupMembers,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App not found or not available"})
		return
	}

	// Use app's max_group_members if not specified or exceeds app limit
	if req.MaxMembers == 0 || req.MaxMembers > app.MaxGroupMembers {
		req.MaxMembers = app.MaxGroupMembers
	}

	// Calculate pricing
	pricePerMember := float64(app.TotalPrice) / float64(req.MaxMembers)
	adminFee := 3500.0 // Fixed admin fee
	totalPrice := float64(app.TotalPrice)

	// Generate unique invite code
	inviteCode := h.generateInviteCode()

	groupID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO groups (
			id, name, description, app_id, owner_id, invite_code, max_members, 
			price_per_member, admin_fee, total_price, group_status, is_public, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, groupID, req.Name, req.Description, req.AppID, userID.(uuid.UUID), inviteCode, req.MaxMembers,
		pricePerMember, adminFee, totalPrice, "open", req.IsPublic, time.Now(), time.Now())

	if err != nil {
		fmt.Printf("Error creating group: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group", "details": err.Error()})
		return
	}

	// Add owner as member
	_, err = h.db.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, joined_at, user_status, payment_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), groupID, userID.(uuid.UUID), "owner", time.Now(), "active", 0)

	if err != nil {
		fmt.Printf("Error adding owner to group: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add owner to group", "details": err.Error()})
		return
	}

	now := time.Now()
	// Get actual member count
	var actualMemberCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM group_members 
		WHERE group_id = $1 AND user_status IN ('active', 'paid')
	`, groupID).Scan(&actualMemberCount)

	if err != nil {
		actualMemberCount = 1 // Fallback to 1 if query fails
	}

	groupResponse := models.GroupResponse{
		ID:             groupID,
		Name:           req.Name,
		Description:    &req.Description,
		AppID:          req.AppID,
		OwnerID:        userID.(uuid.UUID),
		InviteCode:     inviteCode,
		MaxMembers:     req.MaxMembers,
		CurrentMembers: actualMemberCount,
		MemberCount:    actualMemberCount,
		PricePerMember: pricePerMember,
		AdminFee:       adminFee,
		TotalPrice:     totalPrice,
		GroupStatus:    "open",
		IsPublic:       req.IsPublic,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Group created successfully",
		"group":   groupResponse,
	})
}

func (h *GroupHandler) JoinGroup(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.GroupJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find group by invite code
	var group models.Group
	err := h.db.QueryRow(`
		SELECT id, name, description, owner_id, invite_code, max_members, created_at, updated_at
		FROM groups WHERE invite_code = $1
	`, req.InviteCode).Scan(&group.ID, &group.Name, &group.Description, &group.OwnerID, &group.InviteCode, &group.MaxMembers, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid invite code"})
		return
	}

	// Check if group is full
	var currentMemberCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM group_members 
		WHERE group_id = $1 AND user_status IN ('active', 'paid')
	`, group.ID).Scan(&currentMemberCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group capacity"})
		return
	}

	if currentMemberCount >= group.MaxMembers {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group is full"})
		return
	}

	// Check if user is already a member
	var existingMember models.GroupMember
	err = h.db.QueryRow(`
		SELECT id FROM group_members WHERE group_id = $1 AND user_id = $2
	`, group.ID, userID.(uuid.UUID)).Scan(&existingMember.ID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this group"})
		return
	}

	// Check if group is full
	var memberCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM group_members WHERE group_id = $1
	`, group.ID).Scan(&memberCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group capacity"})
		return
	}

	if memberCount >= group.MaxMembers {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group is full"})
		return
	}

	// Add user to group
	_, err = h.db.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, joined_at, user_status, payment_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), group.ID, userID.(uuid.UUID), "member", time.Now(), "pending", 0)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join group"})
		return
	}

	// Update group status based on new member count
	err = h.CheckAndUpdateGroupStatus(group.ID.String())
	if err != nil {
		// Log error but don't fail the join operation
		fmt.Printf("Warning: Failed to update group status: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined group",
		"group":   group,
	})
}

func (h *GroupHandler) GetUserGroups(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	rows, err := h.db.Query(`
		SELECT g.id, g.name, g.description, g.app_id, g.owner_id, g.invite_code, g.max_members, 
		       g.price_per_member, g.admin_fee, g.total_price, g.group_status,
		       g.expires_at, g.created_at, g.updated_at,
		       COUNT(DISTINCT gm.id) as member_count,
		       COUNT(DISTINCT CASE WHEN gm2.user_status IN ('active', 'paid') THEN gm2.id END) as current_members,
		       a.name as app_name, a.description as app_description, a.category, a.icon_url
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		LEFT JOIN group_members gm2 ON g.id = gm2.group_id
		LEFT JOIN apps a ON g.app_id = a.id
		WHERE gm.user_id = $1
		GROUP BY g.id, g.name, g.description, g.app_id, g.owner_id, g.invite_code, g.max_members,
		         g.price_per_member, g.admin_fee, g.total_price, g.group_status,
		         g.expires_at, g.created_at, g.updated_at,
		         a.name, a.description, a.category, a.icon_url
		ORDER BY g.created_at DESC
	`, userID.(uuid.UUID))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}
	defer rows.Close()

	var groups []models.GroupResponse
	for rows.Next() {
		var group models.GroupResponse
		var app models.App
		var appName, appDescription, appCategory, appIconURL *string

		var currentMembers int
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
			&group.InviteCode, &group.MaxMembers, &group.PricePerMember,
			&group.AdminFee, &group.TotalPrice, &group.GroupStatus, &group.ExpiresAt,
			&group.CreatedAt, &group.UpdatedAt, &group.MemberCount, &currentMembers,
			&appName, &appDescription, &appCategory, &appIconURL,
		)

		// Set current_members from real-time count
		group.CurrentMembers = currentMembers
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan group"})
			return
		}

		// Set app information if available
		if appName != nil {
			app.Name = *appName
			if appDescription != nil {
				app.Description = *appDescription
			}
			if appCategory != nil {
				app.Category = *appCategory
			}
			if appIconURL != nil {
				app.IconURL = *appIconURL
			}
			group.App = &app
		}

		groups = append(groups, group)
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// UpdateGroupStatus updates the status of a group
func (h *GroupHandler) UpdateGroupStatus(c *gin.Context) {
	groupID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user is owner of the group
	var ownerID string
	ownerQuery := `SELECT owner_id FROM groups WHERE id = $1`
	err := h.db.QueryRow(ownerQuery, groupID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if ownerID != userID.(uuid.UUID).String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only group owner can update status"})
		return
	}

	// Update group status
	_, err = h.db.Exec(`
		UPDATE groups 
		SET status = $1, updated_at = $2 
		WHERE id = $3
	`, req.Status, time.Now(), groupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group status updated successfully"})
}

// CheckAndUpdateGroupStatus automatically updates group status based on member count
func (h *GroupHandler) CheckAndUpdateGroupStatus(groupID string) error {
	// Get actual member count from group_members table
	var actualMemberCount, maxMembers int
	var currentStatus string
	err := h.db.QueryRow(`
		SELECT 
			COALESCE(COUNT(gm.id), 0) as actual_member_count,
			g.max_members,
			g.group_status
		FROM groups g
		LEFT JOIN group_members gm ON g.id = gm.group_id AND gm.user_status IN ('active', 'paid')
		WHERE g.id = $1
		GROUP BY g.max_members, g.group_status
	`, groupID).Scan(&actualMemberCount, &maxMembers, &currentStatus)

	if err != nil {
		return err
	}

	// Update status based on actual member count
	var newStatus string
	if actualMemberCount >= maxMembers {
		newStatus = "full"
	} else if actualMemberCount > 0 {
		newStatus = "open"
	} else {
		newStatus = "open"
	}

	// Update group_status if needed
	_, err = h.db.Exec(`
		UPDATE groups 
		SET group_status = $1, updated_at = $2 
		WHERE id = $3
	`, newStatus, time.Now(), groupID)

	return err
}

// GetGroupByInviteCode retrieves a group by its invite code
func (h *GroupHandler) GetGroupByInviteCode(c *gin.Context) {
	inviteCode := c.Param("code")
	if inviteCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invite code is required"})
		return
	}

	// Query group with app and owner information
	query := `
		SELECT 
			g.id, g.name, g.description, g.app_id, g.max_members, g.current_members,
			g.price_per_member, g.admin_fee, g.total_price, g.group_status, g.invite_code,
			g.owner_id, g.expires_at, g.created_at, g.updated_at,
			a.name as app_name, a.description as app_description, a.category, a.icon_url,
			u.full_name as owner_name, u.email as owner_email
		FROM groups g
		JOIN apps a ON g.app_id = a.id
		JOIN users u ON g.owner_id = u.id
		WHERE g.invite_code = $1
	`

	var group models.GroupResponse
	var app models.App
	var ownerName, ownerEmail string
	err := h.db.QueryRow(query, inviteCode).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID, &group.MaxMembers,
		&group.CurrentMembers, &group.PricePerMember, &group.AdminFee, &group.TotalPrice,
		&group.GroupStatus, &group.InviteCode, &group.OwnerID, &group.ExpiresAt,
		&group.CreatedAt, &group.UpdatedAt,
		&app.Name, &app.Description, &app.Category, &app.IconURL,
		&ownerName, &ownerEmail,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group"})
		return
	}

	// Set app and owner information
	group.App = &app
	group.Owner = &models.UserResponse{
		FullName: ownerName,
		Email:    ownerEmail,
	}

	c.JSON(http.StatusOK, gin.H{"group": group})
}

func (h *GroupHandler) GetGroupDetails(c *gin.Context) {
	groupID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is member of the group
	var isMember bool
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID.(uuid.UUID)).Scan(&isMember)

	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get group details with app pricing and real-time member count
	var group models.Group
	var appTotalPrice int
	var currentMemberCount int
	err = h.db.QueryRow(`
		SELECT g.id, g.name, g.description, g.app_id, g.owner_id, g.invite_code, g.max_members, 
		       g.price_per_member, g.admin_fee, g.total_price, g.group_status, 
		       g.expires_at, g.created_at, g.updated_at, a.total_price,
		       COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) as current_members
		FROM groups g
		JOIN apps a ON g.app_id = a.id
		LEFT JOIN group_members gm ON g.id = gm.group_id
		WHERE g.id = $1
		GROUP BY g.id, g.name, g.description, g.app_id, g.owner_id, g.invite_code, g.max_members,
		         g.price_per_member, g.admin_fee, g.total_price, g.group_status,
		         g.expires_at, g.created_at, g.updated_at, a.total_price
	`, groupID).Scan(&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
		&group.InviteCode, &group.MaxMembers, &group.PricePerMember,
		&group.AdminFee, &group.TotalPrice, &group.GroupStatus, &group.ExpiresAt, &group.CreatedAt, &group.UpdatedAt, &appTotalPrice, &currentMemberCount)

	// Set current_members from real-time count
	group.CurrentMembers = currentMemberCount

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Recalculate pricing if stored values are 0 (for existing groups)
	if group.PricePerMember == 0 || group.AdminFee == 0 || group.TotalPrice == 0 {
		group.PricePerMember = float64(appTotalPrice) / float64(group.MaxMembers)
		group.AdminFee = 3500.0
		group.TotalPrice = float64(appTotalPrice)

		// Update the group with correct pricing
		_, err = h.db.Exec(`
			UPDATE groups 
			SET price_per_member = $1, admin_fee = $2, total_price = $3, updated_at = $4
			WHERE id = $5
		`, group.PricePerMember, group.AdminFee, group.TotalPrice, time.Now(), groupID)

		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to update group pricing: %v\n", err)
		}
	}

	// Get group members
	memberRows, err := h.db.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.joined_at, gm.user_status, gm.payment_amount,
		       u.email, u.full_name, u.avatar_url
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1
		ORDER BY gm.joined_at ASC
	`, groupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
		return
	}
	defer memberRows.Close()

	var members []models.GroupMember
	for memberRows.Next() {
		var member models.GroupMember
		var user models.UserResponse
		err := memberRows.Scan(&member.ID, &member.GroupID, &member.UserID, &member.JoinedAt, &member.UserStatus, &member.PaymentAmount, &user.Email, &user.FullName, &user.AvatarURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan member"})
			return
		}
		user.ID = member.UserID
		member.User = user
		members = append(members, member)
	}

	group.Members = members

	// Get app information
	var app models.App
	err = h.db.QueryRow(`
		SELECT id, name, description, category, icon_url, total_price
		FROM apps WHERE id = $1
	`, group.AppID).Scan(&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.TotalPrice)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app information"})
		return
	}

	// Get owner information
	var owner models.UserResponse
	err = h.db.QueryRow(`
		SELECT id, email, full_name, avatar_url
		FROM users WHERE id = $1
	`, group.OwnerID).Scan(&owner.ID, &owner.Email, &owner.FullName, &owner.AvatarURL)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch owner information"})
		return
	}

	groupResponse := models.GroupResponse{
		ID:             group.ID,
		Name:           group.Name,
		Description:    group.Description,
		AppID:          group.AppID,
		OwnerID:        group.OwnerID,
		InviteCode:     group.InviteCode,
		MaxMembers:     group.MaxMembers,
		CurrentMembers: group.CurrentMembers,
		PricePerMember: group.PricePerMember,
		AdminFee:       group.AdminFee,
		TotalPrice:     group.TotalPrice,
		GroupStatus:    group.GroupStatus,
		ExpiresAt:      group.ExpiresAt,
		MemberCount:    len(members),
		CreatedAt:      group.CreatedAt,
		UpdatedAt:      group.UpdatedAt,
		Members:        members,
		App:            &app,
		Owner:          &owner,
	}

	c.JSON(http.StatusOK, gin.H{"group": groupResponse})
}

// GetPublicGroups retrieves public groups that users can join
func (h *GroupHandler) GetPublicGroups(c *gin.Context) {
	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	appID := c.Query("app_id") // Get app_id filter

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Query public groups with app information - only show groups that are not full
	// Filter by app_id if provided
	var query string
	var args []interface{}

	if appID != "" {
		query = `
			SELECT 
				g.id, g.name, g.description, g.app_id, g.max_members,
				g.group_status, g.invite_code, g.owner_id, g.is_public, g.expires_at, g.created_at, g.updated_at,
				a.name as app_name, a.description as app_description, a.category, a.icon_url,
				COALESCE(a.total_price, 0) as total_price,
				u.full_name as owner_name,
				-- Calculate pricing dynamically
				(COALESCE(a.total_price, 0) / g.max_members) as price_per_member,
				3500 as admin_fee,
				(COALESCE(a.total_price, 0) / g.max_members) + 3500 as total_per_user,
				COALESCE(a.total_price, 0) as total_price,
				COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) as current_members
			FROM groups g
			JOIN apps a ON g.app_id = a.id
			JOIN users u ON g.owner_id = u.id
			LEFT JOIN group_members gm ON g.id = gm.group_id
			WHERE g.group_status = 'open' AND a.is_active = true 
			AND g.app_id = $1 AND g.is_public = true
			GROUP BY g.id, g.name, g.description, g.app_id, g.max_members,
			         g.group_status, g.invite_code, g.owner_id, g.is_public, g.expires_at, g.created_at, g.updated_at,
			         a.name, a.description, a.category, a.icon_url, a.total_price, u.full_name
			HAVING COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) < g.max_members
			ORDER BY g.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{appID, pageSize, offset}
	} else {
		query = `
			SELECT 
				g.id, g.name, g.description, g.app_id, g.max_members,
				g.group_status, g.invite_code, g.owner_id, g.is_public, g.expires_at, g.created_at, g.updated_at,
				a.name as app_name, a.description as app_description, a.category, a.icon_url,
				COALESCE(a.total_price, 0) as total_price,
				u.full_name as owner_name,
				-- Calculate pricing dynamically
				(COALESCE(a.total_price, 0) / g.max_members) as price_per_member,
				3500 as admin_fee,
				(COALESCE(a.total_price, 0) / g.max_members) + 3500 as total_per_user,
				COALESCE(a.total_price, 0) as total_price,
				COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) as current_members
			FROM groups g
			JOIN apps a ON g.app_id = a.id
			JOIN users u ON g.owner_id = u.id
			LEFT JOIN group_members gm ON g.id = gm.group_id
			WHERE g.group_status = 'open' AND a.is_active = true 
			AND g.is_public = true
			GROUP BY g.id, g.name, g.description, g.app_id, g.max_members,
			         g.group_status, g.invite_code, g.owner_id, g.is_public, g.expires_at, g.created_at, g.updated_at,
			         a.name, a.description, a.category, a.icon_url, a.total_price, u.full_name
			HAVING COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) < g.max_members
			ORDER BY g.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{pageSize, offset}
	}

	fmt.Printf("Executing query: %s\n", query)
	fmt.Printf("With args: %v\n", args)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}
	defer rows.Close()

	var groups []models.GroupResponse
	for rows.Next() {
		var group models.GroupResponse
		var app models.App
		var ownerName string
		var totalPerUser float64
		var currentMembers int
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.AppID, &group.MaxMembers,
			&group.GroupStatus, &group.InviteCode, &group.OwnerID, &group.IsPublic, &group.ExpiresAt,
			&group.CreatedAt, &group.UpdatedAt,
			&app.Name, &app.Description, &app.Category, &app.IconURL,
			&app.TotalPrice,
			&ownerName,
			&group.PricePerMember, &group.AdminFee, &totalPerUser, &group.TotalPrice,
			&currentMembers,
		)

		// Set current_members from real-time count
		group.CurrentMembers = currentMembers
		if err != nil {
			fmt.Printf("Scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan group"})
			return
		}
		// Set MemberCount to CurrentMembers for consistency
		group.MemberCount = group.CurrentMembers
		group.App = &app

		// Set owner information
		group.Owner = &models.UserResponse{
			ID:       group.OwnerID,
			FullName: ownerName,
		}

		groups = append(groups, group)
	}

	fmt.Printf("Found %d groups\n", len(groups))
	fmt.Printf("Groups: %v\n", groups)

	// Get total count with same filters
	var total int
	var countQuery string
	var countArgs []interface{}

	if appID != "" {
		countQuery = `
			SELECT COUNT(*) FROM (
				SELECT g.id
				FROM groups g
				JOIN apps a ON g.app_id = a.id
				LEFT JOIN group_members gm ON g.id = gm.group_id
				WHERE g.group_status = 'open' AND a.is_active = true
				AND g.app_id = $1 AND g.is_public = true
				GROUP BY g.id, g.max_members
				HAVING COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) < g.max_members
			) as filtered_groups
		`
		countArgs = []interface{}{appID}
	} else {
		countQuery = `
			SELECT COUNT(*) FROM (
				SELECT g.id
				FROM groups g
				JOIN apps a ON g.app_id = a.id
				LEFT JOIN group_members gm ON g.id = gm.group_id
				WHERE g.group_status = 'open' AND a.is_active = true
				AND g.is_public = true
				GROUP BY g.id, g.max_members
				HAVING COUNT(DISTINCT CASE WHEN gm.user_status IN ('active', 'paid') THEN gm.id END) < g.max_members
			) as filtered_groups
		`
		countArgs = []interface{}{}
	}

	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups":      groups,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// GetGroupMembers retrieves members of a specific group
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("User not authenticated for group %s", groupID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is member of the group
	var isMember bool
	userIDUUID, ok := userID.(uuid.UUID)
	if !ok {
		log.Printf("Invalid user ID type for group %s", groupID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	log.Printf("Checking membership: user %s for group %s", userIDUUID.String(), groupID)
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userIDUUID).Scan(&isMember)

	if err != nil {
		log.Printf("Error checking membership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check membership"})
		return
	}

	// Debug: return membership status and continue
	log.Printf("User %s membership status for group %s: %v", userIDUUID.String(), groupID, isMember)

	if !isMember {
		log.Printf("User %s is not a member of group %s", userIDUUID.String(), groupID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get group members with price_per_member from groups table
	// Simplified query first to debug
	memberRows, err := h.db.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, 
		       COALESCE(gm.role, 'member') as role, 
		       COALESCE(gm.user_status, 'pending') as user_status, 
		       COALESCE(gm.payment_amount, 0) as payment_amount, 
		       gm.joined_at, 
		       COALESCE(g.price_per_member, 0) as price_per_member,
		       u.full_name, u.email, COALESCE(u.avatar_url, '') as avatar_url
		FROM group_members gm
		LEFT JOIN users u ON gm.user_id = u.id
		LEFT JOIN groups g ON gm.group_id = g.id
		WHERE gm.group_id = $1
		ORDER BY 
			CASE COALESCE(gm.role, 'member')
				WHEN 'owner' THEN 1 
				WHEN 'admin' THEN 2 
				ELSE 3 
			END,
			gm.joined_at ASC
	`, groupID)

	if err != nil {
		log.Printf("Error fetching group members for group %s: %v", groupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
		return
	}
	defer memberRows.Close()

	var members []models.GroupMember
	for memberRows.Next() {
		var member models.GroupMember
		var user models.UserResponse
		var pricePerMember float64
		err := memberRows.Scan(&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.UserStatus, &member.PaymentAmount, &member.JoinedAt, &pricePerMember, &user.FullName, &user.Email, &user.AvatarURL)
		if err != nil {
			log.Printf("Error scanning member: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan member"})
			return
		}
		user.ID = member.UserID
		member.User = user

		// UserStatus already set from database scan

		// Set price_per_member from group
		member.PricePerMember = pricePerMember

		members = append(members, member)
	}

	log.Printf("Found %d members for group %s", len(members), groupID)

	// Get group information with real-time member count
	var group models.Group
	var actualMemberCount int
	err = h.db.QueryRow(`
		SELECT g.id, g.name, g.description, g.app_id, g.max_members,
		       g.price_per_member, g.admin_fee, g.total_price, g.group_status,
		       g.invite_code, g.owner_id, g.expires_at, g.created_at, g.updated_at,
		       COUNT(gm.id) as actual_member_count
		FROM groups g
		LEFT JOIN group_members gm ON g.id = gm.group_id AND gm.user_status IN ('active', 'paid')
		WHERE g.id = $1
		GROUP BY g.id, g.name, g.description, g.app_id, g.max_members,
		         g.price_per_member, g.admin_fee, g.total_price, g.group_status,
		         g.invite_code, g.owner_id, g.expires_at, g.created_at, g.updated_at
	`, groupID).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID,
		&group.MaxMembers, &group.PricePerMember,
		&group.AdminFee, &group.TotalPrice, &group.GroupStatus,
		&group.InviteCode, &group.OwnerID, &group.ExpiresAt,
		&group.CreatedAt, &group.UpdatedAt, &actualMemberCount,
	)

	if err != nil {
		log.Printf("Error fetching group info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group info"})
		return
	}

	// Debug: return more info
	c.JSON(http.StatusOK, gin.H{
		"members": members,
		"group": gin.H{
			"id":               group.ID,
			"name":             group.Name,
			"description":      group.Description,
			"app_id":           group.AppID,
			"max_members":      group.MaxMembers,
			"current_members":  actualMemberCount, // Use real-time count
			"price_per_member": group.PricePerMember,
			"admin_fee":        group.AdminFee,
			"total_price":      group.TotalPrice,
			"group_status":     group.GroupStatus,
			"invite_code":      group.InviteCode,
			"owner_id":         group.OwnerID,
			"expires_at":       group.ExpiresAt,
			"created_at":       group.CreatedAt,
			"updated_at":       group.UpdatedAt,
		},
		"debug": gin.H{
			"group_id":               groupID,
			"user_id":                userIDUUID.String(),
			"member_count":           len(members),
			"actual_member_count":    actualMemberCount,
			"stored_current_members": group.CurrentMembers,
		},
	})
}

// Admin endpoints for state machine management

// AdminUpdateUserStatus allows admin to update any user status
func (h *GroupHandler) AdminUpdateUserStatus(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.AdminUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.stateMachineSvc.AdminUpdateUserStatus(
		adminID.(string),
		req.UserID,
		req.GroupID,
		req.NewStatus,
		req.RemovedReason,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

// AdminUpdateGroupStatus allows admin to update any group status
func (h *GroupHandler) AdminUpdateGroupStatus(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.AdminGroupStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.stateMachineSvc.AdminUpdateGroupStatus(
		adminID.(string),
		req.GroupID,
		req.NewStatus,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group status updated successfully"})
}

// GetUserStatus gets user status in a group
func (h *GroupHandler) GetUserStatus(c *gin.Context) {
	userID := c.Param("user_id")
	groupID := c.Param("group_id")

	var member models.GroupMember
	err := h.db.QueryRow(`
		SELECT id, group_id, user_id, joined_at, status, user_status, 
		       payment_amount, payment_deadline, paid_at, activated_at, 
		       expired_at, removed_at, removed_reason,
		       subscription_period_start, subscription_period_end
		FROM group_members 
		WHERE user_id = $1 AND group_id = $2
	`, userID, groupID).Scan(
		&member.ID, &member.GroupID, &member.UserID, &member.JoinedAt,
		&member.UserStatus, &member.PaymentAmount,
		&member.PaymentDeadline, &member.PaidAt, &member.ActivatedAt,
		&member.ExpiredAt, &member.RemovedAt, &member.RemovedReason,
		&member.SubscriptionPeriodStart, &member.SubscriptionPeriodEnd,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found in group"})
		return
	}

	c.JSON(http.StatusOK, member)
}

// GetGroupStatus gets group status
func (h *GroupHandler) GetGroupStatus(c *gin.Context) {
	groupID := c.Param("group_id")

	var group models.Group
	err := h.db.QueryRow(`
		SELECT id, name, description, app_id, max_members, current_members,
		       price_per_member, admin_fee, total_price, status, group_status,
		       invite_code, owner_id, expires_at, all_paid_at, created_at, updated_at
		FROM groups 
		WHERE id = $1
	`, groupID).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID,
		&group.MaxMembers, &group.CurrentMembers, &group.PricePerMember,
		&group.AdminFee, &group.TotalPrice, &group.GroupStatus,
		&group.InviteCode, &group.OwnerID, &group.ExpiresAt, &group.AllPaidAt,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// LeaveGroup allows a user to leave a group
func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is member of the group
	var isMember bool
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID.(uuid.UUID)).Scan(&isMember)

	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not a member of this group"})
		return
	}

	// Check if user is the owner
	var ownerID string
	err = h.db.QueryRow(`SELECT owner_id FROM groups WHERE id = $1`, groupID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if ownerID == userID.(uuid.UUID).String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group owner cannot leave the group"})
		return
	}

	// Remove user from group
	_, err = h.db.Exec(`
		DELETE FROM group_members 
		WHERE group_id = $1 AND user_id = $2
	`, groupID, userID.(uuid.UUID))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave group"})
		return
	}

	// Update group status based on new member count
	err = h.CheckAndUpdateGroupStatus(groupID)
	if err != nil {
		// Log error but don't fail the leave operation
		fmt.Printf("Warning: Failed to update group status: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully left group",
	})
}

// TransferOwnership - Transfer group ownership to another member
func (h *GroupHandler) TransferOwnership(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groupID := c.Param("id")

	var req struct {
		NewOwnerID string `json:"new_owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if current user is the owner
	var currentOwnerID string
	err := h.db.QueryRow(`
		SELECT owner_id FROM groups WHERE id = $1
	`, groupID).Scan(&currentOwnerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if currentOwnerID != userID.(uuid.UUID).String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only group owner can transfer ownership"})
		return
	}

	// Check if new owner is a member of the group
	var memberExists bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM group_members 
			WHERE group_id = $1 AND user_id = $2 AND user_status IN ('paid', 'active')
		)
	`, groupID, req.NewOwnerID).Scan(&memberExists)

	if err != nil || !memberExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New owner must be an active member of the group"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update group owner
	_, err = tx.Exec(`
		UPDATE groups 
		SET owner_id = $1, updated_at = NOW() 
		WHERE id = $2
	`, req.NewOwnerID, groupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to transfer ownership"})
		return
	}

	// Update member roles - set new owner as owner, old owner as admin
	_, err = tx.Exec(`
		UPDATE group_members 
		SET role = 'owner'
		WHERE group_id = $1 AND user_id = $2
	`, groupID, req.NewOwnerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update new owner role"})
		return
	}

	_, err = tx.Exec(`
		UPDATE group_members 
		SET role = 'admin'
		WHERE group_id = $1 AND user_id = $2
	`, groupID, userID.(uuid.UUID).String())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update old owner role"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete ownership transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ownership transferred successfully",
	})
}

// DeleteGroup - Delete group (set status to removed)
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groupID := c.Param("id")

	// Check if current user is the owner
	var currentOwnerID string
	err := h.db.QueryRow(`
		SELECT owner_id FROM groups WHERE id = $1
	`, groupID).Scan(&currentOwnerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if currentOwnerID != userID.(uuid.UUID).String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only group owner can delete the group"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update group status to removed
	_, err = tx.Exec(`
		UPDATE groups 
		SET group_status = 'closed', updated_at = NOW() 
		WHERE id = $1
	`, groupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	// Delete all members from group
	_, err = tx.Exec(`
		DELETE FROM group_members 
		WHERE group_id = $1
	`, groupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove members"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete group deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Group deleted successfully",
	})
}

// UpdateGroup - Update group details (owner only)
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	groupID := c.Param("id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if current user is the owner
	var currentOwnerID string
	err := h.db.QueryRow(`
		SELECT owner_id FROM groups WHERE id = $1
	`, groupID).Scan(&currentOwnerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if currentOwnerID != userID.(uuid.UUID).String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only group owner can update the group"})
		return
	}

	// Update group
	query := `
		UPDATE groups 
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, description, app_id, owner_id, invite_code, max_members, 
		          current_members, price_per_member, admin_fee, total_price, group_status, 
		          is_public, created_at, updated_at
	`

	var group struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    *string   `json:"description"`
		AppID          string    `json:"app_id"`
		OwnerID        string    `json:"owner_id"`
		InviteCode     string    `json:"invite_code"`
		MaxMembers     int       `json:"max_members"`
		CurrentMembers int       `json:"current_members"`
		PricePerMember float64   `json:"price_per_member"`
		AdminFee       float64   `json:"admin_fee"`
		TotalPrice     float64   `json:"total_price"`
		GroupStatus    string    `json:"group_status"`
		IsPublic       bool      `json:"is_public"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	err = h.db.QueryRow(query, req.Name, req.Description, groupID).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
		&group.InviteCode, &group.MaxMembers, &group.CurrentMembers,
		&group.PricePerMember, &group.AdminFee, &group.TotalPrice,
		&group.GroupStatus, &group.IsPublic, &group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}
