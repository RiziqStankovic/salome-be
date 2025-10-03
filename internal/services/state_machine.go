package services

import (
	"database/sql"
	"fmt"
	"time"

	"salome-be/internal/models"
)

type StateMachineService struct {
	db *sql.DB
}

func NewStateMachineService(db *sql.DB) *StateMachineService {
	return &StateMachineService{db: db}
}

// User State Machine Methods

// UpdateUserStatus updates user status in a group with validation
func (s *StateMachineService) UpdateUserStatus(userID, groupID, newStatus, reason string) error {
	// Validate status transition
	if !s.isValidUserStatusTransition(userID, groupID, newStatus) {
		return fmt.Errorf("invalid status transition")
	}

	now := time.Now()
	var query string
	var args []interface{}

	switch newStatus {
	case models.UserStatusPaid:
		query = `
			UPDATE group_members 
			SET user_status = $1, paid_at = $2, payment_deadline = NULL
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusActive:
		query = `
			UPDATE group_members 
			SET user_status = $1, activated_at = $2
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusExpired:
		query = `
			UPDATE group_members 
			SET user_status = $1, expired_at = $2
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusRemoved:
		query = `
			UPDATE group_members 
			SET user_status = $1, removed_at = $2, removed_reason = $3
			WHERE user_id = $4 AND group_id = $5
		`
		args = []interface{}{newStatus, now, reason, userID, groupID}
	default:
		query = `
			UPDATE group_members 
			SET user_status = $1
			WHERE user_id = $2 AND group_id = $3
		`
		args = []interface{}{newStatus, userID, groupID}
	}

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user status: %v", err)
	}

	// Check if all users are paid and activate group
	if newStatus == models.UserStatusPaid {
		s.checkAndActivateGroup(groupID)
	}

	return nil
}

// isValidUserStatusTransition validates if status transition is allowed
func (s *StateMachineService) isValidUserStatusTransition(userID, groupID, newStatus string) bool {
	var currentStatus string
	err := s.db.QueryRow(`
		SELECT user_status FROM group_members 
		WHERE user_id = $1 AND group_id = $2
	`, userID, groupID).Scan(&currentStatus)

	if err != nil {
		return false
	}

	// Define valid transitions
	validTransitions := map[string][]string{
		models.UserStatusPending: {models.UserStatusPaid, models.UserStatusRemoved},
		models.UserStatusPaid:    {models.UserStatusActive, models.UserStatusRemoved},
		models.UserStatusActive:  {models.UserStatusExpired},
		models.UserStatusExpired: {models.UserStatusActive, models.UserStatusRemoved},
		models.UserStatusRemoved: {}, // End state
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

// checkAndActivateGroup checks if all members are paid and activates the group
func (s *StateMachineService) checkAndActivateGroup(groupID string) error {
	// Count total members and paid members
	var totalMembers, paidMembers int
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN user_status = $1 THEN 1 END) as paid
		FROM group_members 
		WHERE group_id = $2
	`, models.UserStatusPaid, groupID).Scan(&totalMembers, &paidMembers)

	if err != nil {
		return err
	}

	// If all members are paid, activate the group
	if totalMembers > 0 && totalMembers == paidMembers {
		// Use Asia/Jakarta timezone
		loc, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(loc)

		// Update group status
		_, err = s.db.Exec(`
			UPDATE groups 
			SET group_status = $1, all_paid_at = $2
			WHERE id = $3
		`, models.GroupStatusPaidGroup, now, groupID)

		if err != nil {
			return err
		}

		// Activate all members
		_, err = s.db.Exec(`
			UPDATE group_members 
			SET user_status = $1, activated_at = $2
			WHERE group_id = $3 AND user_status = $4
		`, models.UserStatusActive, now, groupID, models.UserStatusPaid)

		if err != nil {
			return err
		}

		// Set subscription period (example: 1 month)
		subscriptionEnd := now.AddDate(0, 1, 0)
		_, err = s.db.Exec(`
			UPDATE group_members 
			SET subscription_period_start = $1, subscription_period_end = $2
			WHERE group_id = $3 AND user_status = $4
		`, now, subscriptionEnd, groupID, models.UserStatusActive)

		return err
	}

	return nil
}

// Group State Machine Methods

// UpdateGroupStatus updates group status with validation
func (s *StateMachineService) UpdateGroupStatus(groupID, newStatus string) error {
	// Validate status transition
	if !s.isValidGroupStatusTransition(groupID, newStatus) {
		return fmt.Errorf("invalid group status transition")
	}

	now := time.Now()
	var query string
	var args []interface{}

	switch newStatus {
	case models.GroupStatusPaidGroup:
		query = `
			UPDATE groups 
			SET group_status = $1, all_paid_at = $2
			WHERE id = $3
		`
		args = []interface{}{newStatus, now, groupID}
	default:
		query = `
			UPDATE groups 
			SET group_status = $1
			WHERE id = $2
		`
		args = []interface{}{newStatus, groupID}
	}

	_, err := s.db.Exec(query, args...)
	return err
}

// isValidGroupStatusTransition validates if group status transition is allowed
func (s *StateMachineService) isValidGroupStatusTransition(groupID, newStatus string) bool {
	var currentStatus string
	err := s.db.QueryRow(`
		SELECT group_status FROM groups WHERE id = $1
	`, groupID).Scan(&currentStatus)

	if err != nil {
		return false
	}

	// Define valid transitions
	validTransitions := map[string][]string{
		models.GroupStatusOpen:      {models.GroupStatusPrivate, models.GroupStatusFull, models.GroupStatusClosed},
		models.GroupStatusPrivate:   {models.GroupStatusOpen, models.GroupStatusFull, models.GroupStatusClosed},
		models.GroupStatusFull:      {models.GroupStatusPaidGroup, models.GroupStatusOpen, models.GroupStatusClosed},
		models.GroupStatusPaidGroup: {models.GroupStatusOpen, models.GroupStatusClosed},
		models.GroupStatusClosed:    {}, // End state
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

// Admin Methods

// IsAdmin checks if user is admin
func (s *StateMachineService) IsAdmin(userID string) (bool, error) {
	var isAdmin bool
	err := s.db.QueryRow(`
		SELECT is_admin FROM users WHERE id = $1
	`, userID).Scan(&isAdmin)

	return isAdmin, err
}

// AdminUpdateUserStatus allows admin to update any user status
func (s *StateMachineService) AdminUpdateUserStatus(adminID, userID, groupID, newStatus, reason string) error {
	// Check if admin
	isAdmin, err := s.IsAdmin(adminID)
	if err != nil {
		return fmt.Errorf("failed to check admin status: %v", err)
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: admin access required")
	}

	// Admin can set any status
	now := time.Now()
	var query string
	var args []interface{}

	switch newStatus {
	case models.UserStatusPaid:
		query = `
			UPDATE group_members 
			SET user_status = $1, paid_at = $2, payment_deadline = NULL
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusActive:
		query = `
			UPDATE group_members 
			SET user_status = $1, activated_at = $2
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusExpired:
		query = `
			UPDATE group_members 
			SET user_status = $1, expired_at = $2
			WHERE user_id = $3 AND group_id = $4
		`
		args = []interface{}{newStatus, now, userID, groupID}
	case models.UserStatusRemoved:
		query = `
			UPDATE group_members 
			SET user_status = $1, removed_at = $2, removed_reason = $3
			WHERE user_id = $4 AND group_id = $5
		`
		args = []interface{}{newStatus, now, reason, userID, groupID}
	default:
		query = `
			UPDATE group_members 
			SET user_status = $1
			WHERE user_id = $2 AND group_id = $3
		`
		args = []interface{}{newStatus, userID, groupID}
	}

	_, err = s.db.Exec(query, args...)
	return err
}

// AdminUpdateGroupStatus allows admin to update any group status
func (s *StateMachineService) AdminUpdateGroupStatus(adminID, groupID, newStatus string) error {
	// Check if admin
	isAdmin, err := s.IsAdmin(adminID)
	if err != nil {
		return fmt.Errorf("failed to check admin status: %v", err)
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: admin access required")
	}

	// Admin can set any status
	now := time.Now()
	var query string
	var args []interface{}

	switch newStatus {
	case models.GroupStatusPaidGroup:
		query = `
			UPDATE groups 
			SET group_status = $1, all_paid_at = $2
			WHERE id = $3
		`
		args = []interface{}{newStatus, now, groupID}
	default:
		query = `
			UPDATE groups 
			SET group_status = $1
			WHERE id = $2
		`
		args = []interface{}{newStatus, groupID}
	}

	_, err = s.db.Exec(query, args...)
	return err
}

// SetPaymentDeadline sets payment deadline for pending users
func (s *StateMachineService) SetPaymentDeadline(userID, groupID string) error {
	deadline := time.Now().Add(time.Duration(models.PaymentTimeoutHours) * time.Hour)

	_, err := s.db.Exec(`
		UPDATE group_members 
		SET payment_deadline = $1
		WHERE user_id = $2 AND group_id = $3 AND user_status = $4
	`, deadline, userID, groupID, models.UserStatusPending)

	return err
}

// CheckExpiredPayments checks for expired payments and removes users
func (s *StateMachineService) CheckExpiredPayments() error {
	now := time.Now()

	// Find users with expired payment deadlines
	rows, err := s.db.Query(`
		SELECT user_id, group_id FROM group_members 
		WHERE user_status = $1 AND payment_deadline < $2
	`, models.UserStatusPending, now)

	if err != nil {
		return err
	}
	defer rows.Close()

	// Remove expired users
	for rows.Next() {
		var userID, groupID string
		if err := rows.Scan(&userID, &groupID); err != nil {
			continue
		}

		// Remove user
		s.UpdateUserStatus(userID, groupID, models.UserStatusRemoved, "Payment timeout")
	}

	return nil
}
