# Phase 1 Implementation Plan - Accounting Foundation

## Current State Review & Next Steps

**Date**: November 4, 2025  
**Phase**: Foundation (Weeks 1-4)  
**Status**: In Progress

---

## Table of Contents

1. [Current Implementation Review](#current-implementation-review)
2. [What's Complete](#whats-complete)
3. [What's Missing](#whats-missing)
4. [Implementation Priorities](#implementation-priorities)
5. [Week-by-Week Plan](#week-by-week-plan)
6. [Technical Specifications](#technical-specifications)

---

## Current Implementation Review

### ✅ Fiscal Year - COMPLETE

You have a **production-ready** fiscal year implementation with:

#### Domain Layer

- **Entity**: `FiscalYear` with all required fields
  - Status workflow: Draft → Open → Closed → Locked
  - Control flags: `IsCurrent`, `IsCalendarYear`, `AllowAdjustingEntries`
  - Financial planning: `BudgetAmount`, `AdjustmentDeadline`
  - Audit fields: `ClosedBy`, `LockedBy`, timestamps
  - Full-text search support

#### Validation Layer

- **Comprehensive validation** via `fiscalyearvalidator`:
  - Date range validation (350-380 days)
  - Calendar year validation (Jan 1 - Dec 31)
  - No overlapping fiscal years
  - Only one current year per organization
  - Status transition rules
  - Tax year validation (IRS compliance)
  - Future year limits (max 5 years ahead)

#### Service Layer

- **Full CRUD operations**:
  - `List()` - with filtering and pagination
  - `Get()` - by ID
  - `GetByYear()` - by year number
  - `GetCurrent()` - get current fiscal year
  - `Create()` - with validation
  - `Update()` - with optimistic locking
  - `Delete()` - with business rule checks
  - `Close()` - year-end closing
  - `Lock()` - permanent lock
  - `Unlock()` - with permission check
  - `Activate()` - set as current year

#### Repository Layer

- **PostgreSQL implementation** with:
  - Efficient queries with proper indexing
  - Tenant isolation (org + business unit)
  - Optimistic locking (version field)
  - Relationship loading (users, etc.)
  - Filter options

#### API Layer

- **RESTful endpoints**:
  - `GET /api/v1/fiscal-years/` - list
  - `POST /api/v1/fiscal-years/` - create
  - `GET /api/v1/fiscal-years/current/` - get current
  - `GET /api/v1/fiscal-years/year/:year/` - get by year
  - `GET /api/v1/fiscal-years/:id/` - get by ID
  - `PUT /api/v1/fiscal-years/:id/` - update
  - `DELETE /api/v1/fiscal-years/:id/` - delete
  - `PUT /api/v1/fiscal-years/:id/close/` - close year
  - `PUT /api/v1/fiscal-years/:id/lock/` - lock year
  - `PUT /api/v1/fiscal-years/:id/unlock/` - unlock year
  - `PUT /api/v1/fiscal-years/:id/activate/` - activate year

#### Frontend

- **React components** (TypeScript):
  - Fiscal year table with sorting/filtering
  - Create modal
  - Edit modal
  - Delete confirmation dialog
  - Form validation
  - Schema validation with Zod

#### Database

- **Migration** complete:
  - `fiscal_years` table
  - `fiscal_year_status_enum` type
  - Proper indexes
  - Foreign key constraints
  - Unique constraints

#### Permissions

- **RBAC integration**:
  - Resource: `fiscal_year`
  - Actions: `read`, `create`, `update`, `delete`, `close`, `lock`, `unlock`, `activate`

---

### ✅ Account Types - COMPLETE

You also have a **production-ready** account type implementation:

#### Domain Layer

- **Entity**: `AccountType`
  - Code (3-10 characters)
  - Name
  - Description
  - Category (Asset, Liability, Equity, Revenue, CostOfRevenue, Expense)
  - Color (hex)
  - IsSystem flag

#### Features

- Full CRUD operations
- Category enum with descriptions
- System account protection
- Full-text search support

---

## What's Complete

### ✅ Fiscal Year (100%)

- [x] Entity design
- [x] Database migration
- [x] Domain validation
- [x] Business rule validation
- [x] Service layer
- [x] Repository layer
- [x] API handlers
- [x] Frontend UI
- [x] Permissions
- [x] Audit logging
- [x] Status workflow
- [x] Year-end closing
- [x] Lock/unlock functionality

### ✅ Account Types (100%)

- [x] Entity design
- [x] Database migration
- [x] Category enum
- [x] Service layer
- [x] Repository layer
- [x] API handlers
- [x] Frontend UI (assumed)
- [x] Permissions

---

## What's Missing

### 🚧 Fiscal Periods (0%)

**Priority**: CRITICAL - Required for Phase 2

Fiscal periods are the monthly/quarterly breakdown of fiscal years. You need these before implementing journal entries.

**Required Features**:

- Auto-generate periods when fiscal year is created (Done)
- Period status (Open/Closed/Locked) (Done)
- Period closing workflow
- Validation (no gaps, no overlaps)
- Cannot post to closed periods

### 🚧 Chart of Accounts (0%)

**Priority**: CRITICAL - Required for Phase 2

The complete list of GL accounts organized hierarchically.

**Required Features**:

- GL Account entity
- Hierarchical structure (parent/child)
- Account code validation
- Account balances (denormalized)
- Trucking-specific default COA
- Bulk import functionality

### 🚧 Year-End Closing Workflow (50%)

**Priority**: HIGH - Enhances fiscal year

You have the basic `Close()` method, but need:

- Pre-close checklist validation
- Closing journal entries generation
- Balance carry-forward to next year
- Opening entries for next year
- Temporal workflow orchestration

### 🚧 Fiscal Year Dashboard (0%)

**Priority**: MEDIUM - Nice to have

A dashboard showing:

- Current fiscal year status
- Period status overview
- Key metrics (budget vs actual)
- Upcoming deadlines
- Action items

---

## Implementation Priorities

### Week 1-2: Fiscal Periods

**Goal**: Enable period-level transaction control

**Tasks**:

1. Create `FiscalPeriod` entity
2. Create database migration
3. Implement auto-generation on fiscal year create
4. Add period closing functionality
5. Add validation rules
6. Create API endpoints
7. Build frontend UI

**Deliverables**:

- Periods auto-generated when fiscal year created
- Period closing workflow
- Cannot post to closed periods (validation ready for Phase 2)

### Week 3-4: Chart of Accounts & GL Accounts

**Goal**: Establish the foundation for all accounting transactions

**Tasks**:

1. Create `GLAccount` entity
2. Create database migration
3. Design trucking-specific COA
4. Implement hierarchical structure
5. Add account balance fields
6. Create bulk import functionality
7. Build API endpoints
8. Build frontend UI (account tree view)

**Deliverables**:

- GL accounts can be created and managed
- Default trucking COA can be imported
- Account hierarchy works
- Ready for journal entries in Phase 2

---

## Week-by-Week Plan

### Week 1: Fiscal Period Entity & Backend

#### Day 1-2: Entity Design & Migration

```go
// File: services/tms/internal/core/domain/accounting/fiscalperiod.go

package accounting

type FiscalPeriod struct {
    bun.BaseModel `bun:"table:fiscal_periods,alias:fp" json:"-"`
    
    ID             pulid.ID       `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
    FiscalYearID   pulid.ID       `json:"fiscalYearId"   bun:"fiscal_year_id,type:VARCHAR(100),notnull"`
    BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
    OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
    
    PeriodNumber   int            `json:"periodNumber"   bun:"period_number,type:INTEGER,notnull"`
    PeriodType     PeriodType     `json:"periodType"     bun:"period_type,type:period_type_enum,notnull,default:'Month'"`
    Name           string         `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
    StartDate      int64          `json:"startDate"      bun:"start_date,type:BIGINT,notnull"`
    EndDate        int64          `json:"endDate"        bun:"end_date,type:BIGINT,notnull"`
    Status         PeriodStatus   `json:"status"         bun:"status,type:period_status_enum,notnull,default:'Open'"`
    
    ClosedAt       *int64         `json:"closedAt"       bun:"closed_at,type:BIGINT,nullzero"`
    ClosedByID     *pulid.ID      `json:"closedById"     bun:"closed_by_id,type:VARCHAR(100),nullzero"`
    
    Version        int64          `json:"version"        bun:"version,type:BIGINT"`
    CreatedAt      int64          `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
    UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
    
    // Relationships
    FiscalYear     *FiscalYear    `json:"fiscalYear,omitempty" bun:"rel:belongs-to,join:fiscal_year_id=id"`
    ClosedBy       *tenant.User   `json:"closedBy,omitempty"   bun:"rel:belongs-to,join:closed_by_id=id"`
}

type PeriodType string
const (
    PeriodTypeMonth   = PeriodType("Month")
    PeriodTypeQuarter = PeriodType("Quarter")
)

type PeriodStatus string
const (
    PeriodStatusOpen   = PeriodStatus("Open")
    PeriodStatusClosed = PeriodStatus("Closed")
    PeriodStatusLocked = PeriodStatus("Locked")
)
```

**Migration**:

```sql
-- File: services/tms/internal/infrastructure/postgres/migrations/YYYYMMDDHHMMSS_add_fiscal_periods.tx.up.sql

CREATE TYPE period_type_enum AS ENUM('Month', 'Quarter');
CREATE TYPE period_status_enum AS ENUM('Open', 'Closed', 'Locked');

CREATE TABLE IF NOT EXISTS "fiscal_periods"(
    "id" varchar(100) NOT NULL,
    "fiscal_year_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "period_number" int NOT NULL,
    "period_type" period_type_enum NOT NULL DEFAULT 'Month',
    "name" varchar(100) NOT NULL,
    "start_date" bigint NOT NULL,
    "end_date" bigint NOT NULL,
    "status" period_status_enum NOT NULL DEFAULT 'Open',
    "closed_at" bigint,
    "closed_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_fiscal_periods" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_periods_fiscal_year" FOREIGN KEY ("fiscal_year_id") REFERENCES "fiscal_years"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_periods_year_number" UNIQUE ("fiscal_year_id", "period_number")
);

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_fiscal_year ON "fiscal_periods"("fiscal_year_id");
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_bu_org ON "fiscal_periods"("business_unit_id", "organization_id");
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_dates ON "fiscal_periods"("start_date", "end_date");
```

#### Day 3-4: Auto-Generation Service

```go
// File: services/tms/internal/core/services/fiscalperiod/generator.go

package fiscalperiod

import (
    "context"
    "time"
)

type Generator struct {
    repo repositories.FiscalPeriodRepository
}

func (g *Generator) GeneratePeriodsForFiscalYear(
    ctx context.Context,
    fiscalYear *accounting.FiscalYear,
) ([]*accounting.FiscalPeriod, error) {
    periods := make([]*accounting.FiscalPeriod, 0, 12)
    
    startTime := time.Unix(fiscalYear.StartDate, 0)
    endTime := time.Unix(fiscalYear.EndDate, 0)
    
    // Generate 12 monthly periods
    currentStart := startTime
    for i := 1; i <= 12; i++ {
        // Calculate period end (last day of month or fiscal year end)
        var periodEnd time.Time
        if i == 12 {
            periodEnd = endTime
        } else {
            periodEnd = currentStart.AddDate(0, 1, 0).Add(-time.Second)
        }
        
        period := &accounting.FiscalPeriod{
            FiscalYearID:   fiscalYear.ID,
            OrganizationID: fiscalYear.OrganizationID,
            BusinessUnitID: fiscalYear.BusinessUnitID,
            PeriodNumber:   i,
            PeriodType:     accounting.PeriodTypeMonth,
            Name:           fmt.Sprintf("Period %d - %s", i, currentStart.Format("January 2006")),
            StartDate:      currentStart.Unix(),
            EndDate:        periodEnd.Unix(),
            Status:         accounting.PeriodStatusOpen,
        }
        
        periods = append(periods, period)
        currentStart = periodEnd.Add(time.Second)
    }
    
    // Bulk insert
    err := g.repo.BulkCreate(ctx, periods)
    if err != nil {
        return nil, err
    }
    
    return periods, nil
}
```

#### Day 5: Update Fiscal Year Service

```go
// File: services/tms/internal/core/services/fiscalyear/service.go

func (s *Service) Create(
    ctx context.Context,
    entity *accounting.FiscalYear,
    userID pulid.ID,
) (*accounting.FiscalYear, error) {
    // ... existing validation ...
    
    createdEntity, err := s.repo.Create(ctx, entity)
    if err != nil {
        return nil, err
    }
    
    // Auto-generate fiscal periods
    if entity.Status == accounting.FiscalYearStatusOpen {
        _, err = s.periodGenerator.GeneratePeriodsForFiscalYear(ctx, createdEntity)
        if err != nil {
            // Log error but don't fail the fiscal year creation
            s.l.Error("failed to generate fiscal periods", zap.Error(err))
        }
    }
    
    // ... existing audit logging ...
    
    return createdEntity, nil
}
```

---

### Week 2: Fiscal Period Frontend & API

#### Day 1-2: API Endpoints

```go
// File: services/tms/internal/api/handlers/fiscalperiod.go

type FiscalPeriodHandler struct {
    service      *fiscalperiodservice.Service
    pm           *middleware.PermissionMiddleware
    errorHandler *helpers.ErrorHandler
}

func (h *FiscalPeriodHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/fiscal-periods/")
    
    // List periods for a fiscal year
    api.GET("", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"), h.list)
    api.GET(":id/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"), h.get)
    
    // Period operations
    api.PUT(":id/close/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "close"), h.close)
    api.PUT(":id/reopen/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "reopen"), h.reopen)
}
```

#### Day 3-5: Frontend Components

```typescript
// File: ui/src/app/fiscal-years/_components/fiscal-period-table.tsx

interface FiscalPeriod {
  id: string;
  periodNumber: number;
  name: string;
  startDate: number;
  endDate: number;
  status: 'Open' | 'Closed' | 'Locked';
}

export function FiscalPeriodTable({ fiscalYearId }: { fiscalYearId: string }) {
  const { data: periods } = useQuery({
    queryKey: ['fiscal-periods', fiscalYearId],
    queryFn: () => getFiscalPeriods(fiscalYearId),
  });
  
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Period</TableHead>
          <TableHead>Name</TableHead>
          <TableHead>Start Date</TableHead>
          <TableHead>End Date</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {periods?.map((period) => (
          <TableRow key={period.id}>
            <TableCell>{period.periodNumber}</TableCell>
            <TableCell>{period.name}</TableCell>
            <TableCell>{formatDate(period.startDate)}</TableCell>
            <TableCell>{formatDate(period.endDate)}</TableCell>
            <TableCell>
              <Badge variant={getStatusVariant(period.status)}>
                {period.status}
              </Badge>
            </TableCell>
            <TableCell>
              {period.status === 'Open' && (
                <Button onClick={() => closePeriod(period.id)}>
                  Close Period
                </Button>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
```

---

### Week 3: GL Account Entity & Backend

#### Day 1-2: Entity Design

```go
// File: services/tms/internal/core/domain/accounting/glaccount.go

package accounting

type GLAccount struct {
    bun.BaseModel `bun:"table:gl_accounts,alias:gla" json:"-"`
    
    ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
    OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
    BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
    AccountTypeID  pulid.ID      `json:"accountTypeId"  bun:"account_type_id,type:VARCHAR(100),notnull"`
    ParentID       *pulid.ID     `json:"parentId"       bun:"parent_id,type:VARCHAR(100),nullzero"`
    
    AccountCode    string        `json:"accountCode"    bun:"account_code,type:VARCHAR(20),notnull"`
    Name           string        `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
    Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
    Category       Category      `json:"category"       bun:"category,type:account_category_enum,notnull"`
    
    // Account Properties
    Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
    IsSystem       bool          `json:"isSystem"       bun:"is_system,type:BOOLEAN,notnull,default:false"`
    AllowManualJE  bool          `json:"allowManualJe"  bun:"allow_manual_je,type:BOOLEAN,notnull,default:true"`
    RequireProject bool          `json:"requireProject" bun:"require_project,type:BOOLEAN,notnull,default:false"`
    
    // Balances (denormalized for performance)
    CurrentBalance int64         `json:"currentBalance" bun:"current_balance,type:BIGINT,notnull,default:0"`
    DebitBalance   int64         `json:"debitBalance"   bun:"debit_balance,type:BIGINT,notnull,default:0"`
    CreditBalance  int64         `json:"creditBalance"  bun:"credit_balance,type:BIGINT,notnull,default:0"`
    
    Version        int64         `json:"version"        bun:"version,type:BIGINT"`
    CreatedAt      int64         `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
    UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
    SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
    Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
    
    // Relationships
    AccountType    *AccountType  `json:"accountType,omitempty" bun:"rel:belongs-to,join:account_type_id=id"`
    Parent         *GLAccount    `json:"parent,omitempty"      bun:"rel:belongs-to,join:parent_id=id"`
    Children       []*GLAccount  `json:"children,omitempty"    bun:"rel:has-many,join:id=parent_id"`
}
```

**Migration**:

```sql
CREATE TABLE IF NOT EXISTS "gl_accounts"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "account_type_id" varchar(100) NOT NULL,
    "parent_id" varchar(100),
    "account_code" varchar(20) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "category" account_category_enum NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "is_system" boolean NOT NULL DEFAULT false,
    "allow_manual_je" boolean NOT NULL DEFAULT true,
    "require_project" boolean NOT NULL DEFAULT false,
    "current_balance" bigint NOT NULL DEFAULT 0,
    "debit_balance" bigint NOT NULL DEFAULT 0,
    "credit_balance" bigint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_gl_accounts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_gl_accounts_account_type" FOREIGN KEY ("account_type_id") REFERENCES "account_types"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_gl_accounts_parent" FOREIGN KEY ("parent_id") REFERENCES "gl_accounts"("id") ON DELETE RESTRICT,
    CONSTRAINT "uq_gl_accounts_code" UNIQUE ("organization_id", "account_code")
);

CREATE INDEX IF NOT EXISTS idx_gl_accounts_account_type ON "gl_accounts"("account_type_id");
CREATE INDEX IF NOT EXISTS idx_gl_accounts_parent ON "gl_accounts"("parent_id");
CREATE INDEX IF NOT EXISTS idx_gl_accounts_code ON "gl_accounts"("account_code");
CREATE INDEX IF NOT EXISTS idx_gl_accounts_category ON "gl_accounts"("category");
```

#### Day 3-5: Default COA Seed Data

```go
// File: services/tms/internal/infrastructure/database/seeds/accounting/01_default_coa.go

package accounting

var DefaultTruckingCOA = []GLAccountSeed{
    // ASSETS (1000-1999)
    {Code: "1000", Name: "Cash and Cash Equivalents", Category: "Asset", IsSystem: true},
    {Code: "1010", Name: "Operating Cash", Category: "Asset", Parent: "1000"},
    {Code: "1020", Name: "Payroll Cash", Category: "Asset", Parent: "1000"},
    {Code: "1030", Name: "Fuel Card Cash", Category: "Asset", Parent: "1000"},
    
    {Code: "1100", Name: "Accounts Receivable", Category: "Asset", IsSystem: true},
    {Code: "1110", Name: "AR - Trade", Category: "Asset", Parent: "1100"},
    {Code: "1120", Name: "AR - Fuel Surcharge", Category: "Asset", Parent: "1100"},
    {Code: "1190", Name: "Allowance for Doubtful Accounts", Category: "Asset", Parent: "1100"},
    
    // ... (complete COA from roadmap)
    
    // REVENUE (4000-4999)
    {Code: "4000", Name: "Freight Revenue", Category: "Revenue", IsSystem: true},
    {Code: "4010", Name: "Linehaul Revenue", Category: "Revenue", Parent: "4000"},
    {Code: "4020", Name: "Fuel Surcharge Revenue", Category: "Revenue", Parent: "4000"},
    {Code: "4030", Name: "Accessorial Revenue", Category: "Revenue", Parent: "4000"},
    
    // COST OF REVENUE (5000-5999)
    {Code: "5000", Name: "Driver Costs", Category: "CostOfRevenue", IsSystem: true},
    {Code: "5010", Name: "Driver Wages - Company", Category: "CostOfRevenue", Parent: "5000"},
    {Code: "5020", Name: "Driver Wages - Owner Operator", Category: "CostOfRevenue", Parent: "5000"},
    
    {Code: "5100", Name: "Fuel Costs", Category: "CostOfRevenue", IsSystem: true},
    {Code: "5110", Name: "Diesel Fuel", Category: "CostOfRevenue", Parent: "5100"},
    {Code: "5120", Name: "DEF (Diesel Exhaust Fluid)", Category: "CostOfRevenue", Parent: "5100"},
    
    // ... (complete COA)
}
```

---

### Week 4: GL Account Frontend & COA Import

#### Day 1-2: API Endpoints

```go
// File: services/tms/internal/api/handlers/glaccount.go

func (h *GLAccountHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/gl-accounts/")
    
    api.GET("", h.pm.RequirePermission(permission.ResourceGLAccount, "read"), h.list)
    api.GET("tree/", h.pm.RequirePermission(permission.ResourceGLAccount, "read"), h.getTree)
    api.GET(":id/", h.pm.RequirePermission(permission.ResourceGLAccount, "read"), h.get)
    api.POST("", h.pm.RequirePermission(permission.ResourceGLAccount, "create"), h.create)
    api.PUT(":id/", h.pm.RequirePermission(permission.ResourceGLAccount, "update"), h.update)
    api.DELETE(":id/", h.pm.RequirePermission(permission.ResourceGLAccount, "delete"), h.delete)
    
    // Bulk operations
    api.POST("bulk-import/", h.pm.RequirePermission(permission.ResourceGLAccount, "create"), h.bulkImport)
    api.POST("import-default-coa/", h.pm.RequirePermission(permission.ResourceGLAccount, "create"), h.importDefaultCOA)
    
    // Balance operations
    api.GET(":id/balance/", h.pm.RequirePermission(permission.ResourceGLAccount, "read"), h.getBalance)
    api.GET(":id/activity/", h.pm.RequirePermission(permission.ResourceGLAccount, "read"), h.getActivity)
}
```

#### Day 3-5: Frontend Components

```typescript
// File: ui/src/app/chart-of-accounts/_components/account-tree.tsx

export function AccountTree() {
  const { data: accounts } = useQuery({
    queryKey: ['gl-accounts', 'tree'],
    queryFn: () => getGLAccountsTree(),
  });
  
  return (
    <div className="space-y-2">
      <div className="flex justify-between items-center">
        <h2>Chart of Accounts</h2>
        <div className="space-x-2">
          <Button onClick={() => importDefaultCOA()}>
            Import Default COA
          </Button>
          <Button onClick={() => openCreateModal()}>
            <Plus className="h-4 w-4 mr-2" />
            New Account
          </Button>
        </div>
      </div>
      
      <div className="border rounded-lg">
        {accounts?.map((category) => (
          <AccountCategory key={category.name} category={category}>
            {category.accounts.map((account) => (
              <AccountNode key={account.id} account={account} />
            ))}
          </AccountCategory>
        ))}
      </div>
    </div>
  );
}

function AccountNode({ account }: { account: GLAccount }) {
  const [isExpanded, setIsExpanded] = useState(false);
  
  return (
    <div className="pl-4">
      <div className="flex items-center justify-between p-2 hover:bg-gray-50">
        <div className="flex items-center space-x-2">
          {account.children?.length > 0 && (
            <button onClick={() => setIsExpanded(!isExpanded)}>
              {isExpanded ? <ChevronDown /> : <ChevronRight />}
            </button>
          )}
          <span className="font-mono text-sm">{account.accountCode}</span>
          <span>{account.name}</span>
          {account.isSystem && (
            <Badge variant="secondary">System</Badge>
          )}
        </div>
        <div className="flex items-center space-x-4">
          <span className="font-mono text-sm">
            {formatCurrency(account.currentBalance)}
          </span>
          <AccountActions account={account} />
        </div>
      </div>
      
      {isExpanded && account.children?.map((child) => (
        <AccountNode key={child.id} account={child} />
      ))}
    </div>
  );
}
```

---

## Technical Specifications

### Database Schema Summary

After Phase 1 completion, you'll have:

```
fiscal_years (existing)
├── fiscal_periods (new)
│   └── FK: fiscal_year_id → fiscal_years.id
│
account_types (existing)
└── gl_accounts (new)
    ├── FK: account_type_id → account_types.id
    └── FK: parent_id → gl_accounts.id (self-reference)
```

### API Endpoint Summary

```
Fiscal Years (existing):
  GET    /api/v1/fiscal-years/
  POST   /api/v1/fiscal-years/
  GET    /api/v1/fiscal-years/current/
  GET    /api/v1/fiscal-years/year/:year/
  GET    /api/v1/fiscal-years/:id/
  PUT    /api/v1/fiscal-years/:id/
  DELETE /api/v1/fiscal-years/:id/
  PUT    /api/v1/fiscal-years/:id/close/
  PUT    /api/v1/fiscal-years/:id/lock/
  PUT    /api/v1/fiscal-years/:id/unlock/
  PUT    /api/v1/fiscal-years/:id/activate/

Fiscal Periods (new):
  GET    /api/v1/fiscal-periods/
  GET    /api/v1/fiscal-periods/:id/
  PUT    /api/v1/fiscal-periods/:id/close/
  PUT    /api/v1/fiscal-periods/:id/reopen/

GL Accounts (new):
  GET    /api/v1/gl-accounts/
  GET    /api/v1/gl-accounts/tree/
  GET    /api/v1/gl-accounts/:id/
  POST   /api/v1/gl-accounts/
  PUT    /api/v1/gl-accounts/:id/
  DELETE /api/v1/gl-accounts/:id/
  POST   /api/v1/gl-accounts/bulk-import/
  POST   /api/v1/gl-accounts/import-default-coa/
  GET    /api/v1/gl-accounts/:id/balance/
  GET    /api/v1/gl-accounts/:id/activity/
```

---

## Success Criteria

### Phase 1 Complete When

- [x] Fiscal years fully functional (DONE)
- [ ] Fiscal periods auto-generated on fiscal year create
- [ ] Periods can be closed in order
- [ ] Chart of accounts created with trucking-specific structure
- [ ] GL accounts can be created, edited, deleted
- [ ] Account hierarchy works (parent/child)
- [ ] Default COA can be imported
- [ ] Account balances tracked (ready for journal entries)
- [ ] All APIs tested and working
- [ ] Frontend UI complete and polished
- [ ] Documentation updated

---

## Next Steps

### Immediate Actions (This Week)

1. **Review this document** - Make sure you agree with the plan
2. **Create fiscal period entity** - Start with domain layer
3. **Create fiscal period migration** - Database schema
4. **Implement auto-generation** - Generate periods on fiscal year create

### Week 2 Actions

1. **Complete fiscal period backend**
2. **Build fiscal period API**
3. **Create fiscal period UI**
4. **Test period closing workflow**

### Week 3-4 Actions

1. **Create GL account entity**
2. **Design default trucking COA**
3. **Implement account hierarchy**
4. **Build GL account UI**
5. **Test COA import**

---

## Questions to Answer

Before we start implementation, let's discuss:

1. **Fiscal Period Generation**:
   - Should we generate periods automatically when fiscal year is created?
   - Or should it be a manual action (button click)?
   - **Recommendation**: Auto-generate when status changes to "Open"

2. **Period Type**:
   - Do you want to support quarterly periods?
   - Or just monthly (12 periods)?
   - **Recommendation**: Start with monthly only, add quarterly later

3. **Chart of Accounts**:
   - Should we import the default COA automatically for new organizations?
   - Or require manual import?
   - **Recommendation**: Show a wizard on first login to import default COA

4. **Account Numbering**:
   - Do you want to enforce a specific numbering scheme?
   - Or allow flexible account codes?
   - **Recommendation**: Flexible codes, but provide default structure

5. **Account Balances**:
   - Should we calculate balances in real-time?
   - Or denormalize and update on journal entry posting?
   - **Recommendation**: Denormalize for performance (update on post)

---

## Conclusion

You have an **excellent foundation** with your fiscal year implementation. It's production-ready and follows best practices. Now we need to:

1. **Add fiscal periods** - Enable period-level control
2. **Add GL accounts** - Establish the chart of accounts
3. **Prepare for Phase 2** - Journal entries and posting engine

The good news is that your existing architecture is solid, so adding these features will be straightforward. The patterns you've established (domain, service, repository, API, frontend) are consistent and well-designed.

Let me know if you want to:

- **Start implementing fiscal periods** (I can write the code)
- **Review the GL account design** (make sure it fits your needs)
- **Discuss any of the questions above**
- **Something else**

What would you like to tackle first? 🚀
