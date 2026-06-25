export default function StatCard({ title, value, color }: { title: string; value: number | string; color?: string }) {
  return (
    <div className="stat-card">
      <div className="stat-value" style={color ? { color } : undefined}>{value}</div>
      <div className="stat-label">{title}</div>
    </div>
  );
}
