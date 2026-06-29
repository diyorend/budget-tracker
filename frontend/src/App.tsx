import { useState, useEffect } from "react";
import { Toaster } from "react-hot-toast";
import { login, register } from "./api/auth";
import {
  getTransactions,
  createTransaction,
  getBudgetStatus,
  upsertBudget,
} from "./api/transactions";
import type { Transaction, BudgetStatus } from "./api/transactions";
import { SpendingChart } from "./components/SpendingChart";
import { useAlertWebSocket } from "./hooks/useWebSocket";

export default function App() {
  const [token, setToken] = useState<string | null>(
    localStorage.getItem("token"),
  );
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [budgets, setBudgets] = useState<BudgetStatus[]>([]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [isRegistering, setIsRegistering] = useState(false);

  // Transaction form state
  const [txAmount, setTxAmount] = useState("");
  const [txCategory, setTxCategory] = useState("Food");
  const [txDesc, setTxDesc] = useState("");

  // Budget form state
  const [budgetCategory, setBudgetCategory] = useState("Food");
  const [budgetLimit, setBudgetLimit] = useState("");

  useAlertWebSocket();

  useEffect(() => {
    if (token) {
      loadData();
    }
  }, [token]);

  const loadData = async () => {
    const [txRes, budgetRes] = await Promise.all([
      getTransactions(),
      getBudgetStatus(),
    ]);
    setTransactions(txRes.data);
    setBudgets(budgetRes.data);
  };

  const handleAuth = async () => {
    try {
      const fn = isRegistering ? register : login;
      const res = await fn(email, password);
      localStorage.setItem("token", res.data.token);
      setToken(res.data.token);
    } catch (e: any) {
      alert(e.response?.data?.error || "Auth failed");
    }
  };

  const handleAddTransaction = async () => {
    if (!txAmount || !txCategory) return;
    await createTransaction({
      amount: parseFloat(txAmount),
      category: txCategory,
      description: txDesc,
    });
    setTxAmount("");
    setTxDesc("");
    await loadData();
  };

  const handleSetBudget = async () => {
    if (!budgetLimit || !budgetCategory) return;
    await upsertBudget({
      category: budgetCategory,
      limit_amount: parseFloat(budgetLimit),
    });
    setBudgetLimit("");
    await loadData();
  };

  if (!token) {
    return (
      <div
        style={{
          maxWidth: 400,
          margin: "80px auto",
          padding: 24,
          fontFamily: "system-ui",
        }}
      >
        <Toaster />
        <h1 style={{ marginBottom: 24 }}>Budget Tracker</h1>
        <input
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          style={inputStyle}
        />
        <input
          placeholder="Password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          style={inputStyle}
        />
        <button onClick={handleAuth} style={btnStyle}>
          {isRegistering ? "Register" : "Login"}
        </button>
        <button
          onClick={() => setIsRegistering(!isRegistering)}
          style={{
            ...btnStyle,
            background: "none",
            color: "#6b7280",
            marginTop: 8,
          }}
        >
          {isRegistering
            ? "Already have account? Login"
            : "No account? Register"}
        </button>
      </div>
    );
  }

  const CATEGORIES = [
    "Food",
    "Transport",
    "Entertainment",
    "Health",
    "Shopping",
    "Other",
  ];

  return (
    <div
      style={{
        maxWidth: 900,
        margin: "0 auto",
        padding: 24,
        fontFamily: "system-ui",
      }}
    >
      <Toaster position="top-right" />
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 32,
        }}
      >
        <h1>Budget Tracker</h1>
        <button
          onClick={() => {
            localStorage.removeItem("token");
            setToken(null);
          }}
          style={{ ...btnStyle, background: "#6b7280", padding: "8px 16px" }}
        >
          Logout
        </button>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 24,
          marginBottom: 32,
        }}
      >
        {/* Add Transaction */}
        <div style={cardStyle}>
          <h2 style={{ marginTop: 0 }}>Add Transaction</h2>
          <input
            placeholder="Amount"
            type="number"
            value={txAmount}
            onChange={(e) => setTxAmount(e.target.value)}
            style={inputStyle}
          />
          <select
            value={txCategory}
            onChange={(e) => setTxCategory(e.target.value)}
            style={inputStyle}
          >
            {CATEGORIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
          <input
            placeholder="Description (optional)"
            value={txDesc}
            onChange={(e) => setTxDesc(e.target.value)}
            style={inputStyle}
          />
          <button onClick={handleAddTransaction} style={btnStyle}>
            Add
          </button>
        </div>

        {/* Set Budget */}
        <div style={cardStyle}>
          <h2 style={{ marginTop: 0 }}>Set Budget</h2>
          <select
            value={budgetCategory}
            onChange={(e) => setBudgetCategory(e.target.value)}
            style={inputStyle}
          >
            {CATEGORIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
          <input
            placeholder="Monthly limit"
            type="number"
            value={budgetLimit}
            onChange={(e) => setBudgetLimit(e.target.value)}
            style={inputStyle}
          />
          <button onClick={handleSetBudget} style={btnStyle}>
            Save Budget
          </button>
        </div>
      </div>

      {/* Spending Chart */}
      {budgets.length > 0 && (
        <div style={cardStyle}>
          <h2 style={{ marginTop: 0 }}>Spending vs Budget</h2>
          <SpendingChart budgets={budgets} />
        </div>
      )}

      {/* Budget Status Cards */}
      {budgets.length > 0 && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, 1fr)",
            gap: 16,
            margin: "24px 0",
          }}
        >
          {budgets.map((b) => (
            <div
              key={b.id}
              style={{
                ...cardStyle,
                borderLeft: `4px solid ${b.percentage >= 100 ? "#ef4444" : b.percentage >= 80 ? "#f59e0b" : "#22c55e"}`,
              }}
            >
              <strong>{b.category}</strong>
              <div style={{ fontSize: 13, color: "#6b7280", marginTop: 4 }}>
                ${Number(b.spent).toFixed(2)} / $
                {Number(b.limit_amount).toFixed(2)}
              </div>
              <div
                style={{
                  marginTop: 8,
                  background: "#e5e7eb",
                  borderRadius: 4,
                  height: 6,
                }}
              >
                <div
                  style={{
                    width: `${Math.min(b.percentage, 100)}%`,
                    height: "100%",
                    background:
                      b.percentage >= 100
                        ? "#ef4444"
                        : b.percentage >= 80
                          ? "#f59e0b"
                          : "#22c55e",
                    borderRadius: 4,
                  }}
                />
              </div>
              <div style={{ fontSize: 12, marginTop: 4, color: "#6b7280" }}>
                {b.percentage.toFixed(0)}% used
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Recent Transactions */}
      <div style={cardStyle}>
        <h2 style={{ marginTop: 0 }}>Recent Transactions</h2>
        {transactions.length === 0 && (
          <p style={{ color: "#9ca3af" }}>No transactions yet</p>
        )}
        {transactions.map((t) => (
          <div
            key={t.id}
            style={{
              display: "flex",
              justifyContent: "space-between",
              padding: "12px 0",
              borderBottom: "1px solid #f3f4f6",
            }}
          >
            <div>
              <strong>{t.category}</strong>
              {t.description && (
                <span style={{ color: "#6b7280", marginLeft: 8 }}>
                  {t.description}
                </span>
              )}
              <div style={{ fontSize: 12, color: "#9ca3af" }}>
                {new Date(t.date).toLocaleDateString()}
              </div>
            </div>
            <strong style={{ color: "#ef4444" }}>
              -${Number(t.amount).toFixed(2)}
            </strong>
          </div>
        ))}
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  width: "100%",
  padding: "10px 12px",
  marginBottom: 12,
  border: "1px solid #d1d5db",
  borderRadius: 6,
  fontSize: 14,
  boxSizing: "border-box",
};
const btnStyle: React.CSSProperties = {
  width: "100%",
  padding: "10px 0",
  background: "#3b82f6",
  color: "#fff",
  border: "none",
  borderRadius: 6,
  fontSize: 14,
  cursor: "pointer",
  fontWeight: 600,
};
const cardStyle: React.CSSProperties = {
  background: "#fff",
  border: "1px solid #e5e7eb",
  borderRadius: 8,
  padding: 20,
  boxShadow: "0 1px 3px rgba(0,0,0,0.05)",
};
