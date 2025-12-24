package domain

import "time"

// Enumerations
const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleStaff   UserRole = "staff"

	LogInfo    ActivityLogType = "info"
	LogWarning ActivityLogType = "warning"
	LogError   ActivityLogType = "error"

	AttendancePresent AttendanceStatus = "present"
	AttendanceLeave   AttendanceStatus = "leave"
	AttendanceSick    AttendanceStatus = "sick"
	AttendanceOff     AttendanceStatus = "off"

	TransactionPaid   TransactionStatus = "paid"
	TransactionRefund TransactionStatus = "refund"

	FinanceRevenue FinanceEntryType = "revenue"
	FinanceExpense FinanceEntryType = "expense"

	NotificationInfo    NotificationType = "info"
	NotificationWarning NotificationType = "warning"
	NotificationError   NotificationType = "error"
)

type UserRole string
type ActivityLogType string
type AttendanceStatus string
type TransactionStatus string
type FinanceEntryType string
type NotificationType string

type Money struct {
	Amount   int64
	Currency string
}

type Region struct {
	ID        int64
	TenantID  *int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type User struct {
	ID           int64
	TenantID     *int64
	Name         string
	Email        string
	Region       string
	Phone        string
	Address      string
	RegionID     *int64
	Role         UserRole
	IsGoogle     bool
	PasswordHash *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type Session struct {
	ID           int64
	TenantID     *int64
	UserID       *int64
	Token        *string
	RefreshToken *string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

type ActivityLog struct {
	ID        int64
	TenantID  *int64
	Title     string
	Message   string
	Actor     string
	Type      ActivityLogType
	LoggedAt  time.Time
	Synced    bool
	DeletedAt *time.Time
}

type Settings struct {
	TenantID             *int64
	BusinessName         string
	BusinessAddress      string
	BusinessPhone        string
	ReceiptFooter        string
	DefaultPaymentMethod string
	PrinterName          string
	PaperSize            string
	AutoPrint            bool
	Notifications        bool
	TrackStock           bool
	RoundingPrice        bool
	AutoBackup           bool
	CashierPin           bool
	CurrencyCode         string
	UpdatedAt            time.Time
}

type Category struct {
	ID        int64
	TenantID  *int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Customer struct {
	ID        int64
	TenantID  *int64
	Name      string
	Phone     string
	Email     string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Product struct {
	ID         int64
	TenantID   *int64
	Name       string
	Category   string
	CategoryID int64
	Price      Money
	Image      string
	TrackStock bool
	Stock      int
	MinStock   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type Stock struct {
	ID           int64
	TenantID     *int64
	ProductID    *int64
	Name         string
	Category     string
	Image        string
	Stock        int
	Transactions int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type Employee struct {
	ID         int64
	TenantID   *int64
	ManagerID  *int64
	Name       string
	Role       string
	Phone      string
	Email      string
	PinHash    *string
	JoinDate   time.Time
	Commission *float64
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type Attendance struct {
	ID           int64
	TenantID     *int64
	EmployeeID   *int64
	EmployeeName string
	Date         time.Time
	CheckIn      *time.Time
	CheckOut     *time.Time
	Status       AttendanceStatus
	Source       string
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

type FinanceEntry struct {
	ID        int64
	TenantID  *int64
	Title     string
	Amount    Money
	Category  string
	Date      time.Time
	Type      FinanceEntryType
	Note      string
	TransactionCode *string
	Staff     *string
	Service   *string
	CreatedAt time.Time
	DeletedAt *time.Time
}

type MembershipState struct {
	TenantID  *int64
	OwnerID   *int64
	UsedQuota int
	FreeUsed  int
	FreeStart time.Time
	TopupBal  int
	UpdatedAt time.Time
}

type MembershipTopup struct {
	ID        int64
	TenantID  *int64
	OwnerID   *int64
	Amount    Money
	Manager   string
	Note      string
	Date      time.Time
	CreatedAt time.Time
	DeletedAt *time.Time
}

type Notification struct {
	ID        int64
	UserID    *int64
	Title     string
	Message   string
	Type      NotificationType
	CreatedAt time.Time
	ReadAt    *time.Time
	DeletedAt *time.Time
}

type Transaction struct {
	ID               int64
	TenantID         *int64
	Code             string
	Date             time.Time
	Time             string
	Amount           Money
	PaymentMethod    string
	ShiftID          *string
	OperatorName     string
	StylistID        *int64
	PaymentIntentID  *string
	PaymentReference *string
	Status           TransactionStatus
	RefundedAt       *time.Time
	RefundedBy       *int64
	RefundNote       string
	Stylist          string
	CustomerID       *int64
	Customer         *TransactionCustomerSnapshot
	Items            []TransactionItem
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TransactionCustomerSnapshot struct {
	Name      string
	Phone     string
	Email     string
	Address   string
	Visits    *int
	LastVisit *string
}

type TransactionItem struct {
	ID            int64
	TenantID      *int64
	TransactionID int64
	ProductID     *int64
	Name          string
	Category      string
	CategoryID    *int64
	Price         Money
	Qty           int
	CreatedAt     time.Time
	DeletedAt     *time.Time
}

type ClosingHistory struct {
	ID           int64
	TenantID     *int64
	Date         time.Time
	Shift        string
	Karyawan     string
	ShiftID      *string
	OperatorName string
	Total        Money
	Status       string
	Catatan      string
	Fisik        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
