CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    limit_amount NUMERIC(12, 2) NOT NULL CHECK (limit_amount > 0),
    month DATE NOT NULL, -- store as first day of month
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category, month)
);

CREATE INDEX idx_budgets_user_month ON budgets(user_id, month);