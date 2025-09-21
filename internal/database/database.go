package database

import (
	"database/sql"
	"fmt"
	"log"

	"salome-be/internal/config"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	host := config.GetEnv("DB_HOST", "localhost")
	port := config.GetEnv("DB_PORT", "5432")
	user := config.GetEnv("DB_USER", "salome_user")
	password := config.GetEnv("DB_PASSWORD", "salome_password")
	dbname := config.GetEnv("DB_NAME", "salome_db")
	sslmode := config.GetEnv("DB_SSLMODE", "disable")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Database connected successfully")
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	// Create users table
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		full_name VARCHAR(255) NOT NULL,
		avatar_url VARCHAR(500),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Create groups table
	groupsTable := `
	CREATE TABLE IF NOT EXISTS groups (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		invite_code VARCHAR(10) UNIQUE NOT NULL,
		max_members INTEGER DEFAULT 10,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Create group_members table
	groupMembersTable := `
	CREATE TABLE IF NOT EXISTS group_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_id, user_id)
	);`

	// Create subscriptions table
	subscriptionsTable := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
		service_name VARCHAR(255) NOT NULL,
		service_url VARCHAR(500),
		plan_name VARCHAR(255) NOT NULL,
		price_per_month DECIMAL(10,2) NOT NULL,
		currency VARCHAR(3) DEFAULT 'IDR',
		status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled')),
		next_billing_date DATE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Create subscription_shares table
	subscriptionSharesTable := `
	CREATE TABLE IF NOT EXISTS subscription_shares (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		share_percentage DECIMAL(5,2) NOT NULL,
		amount DECIMAL(10,2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(subscription_id, user_id)
	);`

	// Create payments table
	paymentsTable := `
	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		amount DECIMAL(10,2) NOT NULL,
		currency VARCHAR(3) DEFAULT 'IDR',
		status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
		midtrans_transaction_id VARCHAR(255),
		payment_method VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Create account_credentials table
	accountCredentialsTable := `
	CREATE TABLE IF NOT EXISTS account_credentials (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
		username VARCHAR(255),
		password_encrypted TEXT,
		additional_info JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	tables := []string{
		usersTable,
		groupsTable,
		groupMembersTable,
		subscriptionsTable,
		subscriptionSharesTable,
		paymentsTable,
		accountCredentialsTable,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}
