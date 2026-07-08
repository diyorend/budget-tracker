import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import type { BudgetStatus } from "../api/transactions";

interface Props {
  budgets: BudgetStatus[];
}

export function SpendingChart({ budgets }: Props) {
  const data = budgets.map((b) => ({
    name: b.category,
    spent: Number(b.spent),
    limit: Number(b.limit_amount),
    percentage: b.percentage,
  }));

  return (
    <div style={{ width: "100%", height: 280 }}>
      <ResponsiveContainer>
        <BarChart
          data={data}
          margin={{ top: 5, right: 20, left: 0, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip formatter={(value) => `$${Number(value).toFixed(2)}`} />
          <Bar dataKey="limit" fill="#e5e7eb" name="Budget" />
          <Bar dataKey="spent" name="Spent">
            {data.map((entry, index) => (
              <Cell
                key={index}
                fill={
                  entry.percentage >= 100
                    ? "#ef4444"
                    : entry.percentage >= 80
                      ? "#f59e0b"
                      : "#22c55e"
                }
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
