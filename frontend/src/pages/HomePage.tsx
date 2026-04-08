import { Link } from "react-router-dom";

export default function HomePage() {
  return (
    <div style={{
      display: "flex", flexDirection: "column", alignItems: "center",
      justifyContent: "center", minHeight: "60vh", textAlign: "center",
      padding: "48px 24px",
    }}>
      <h1 style={{ fontSize: "2.2rem", fontWeight: 800, marginBottom: 12 }}>
        Определение видимости МКС
      </h1>
      <p style={{ color: "#B0BEC5", fontSize: "1.05rem", maxWidth: 600, marginBottom: 32, lineHeight: 1.7 }}>
        Выберите точки наблюдения на Земле и создайте заявку на определение
        положения Международной космической станции.
      </p>
      <Link
        to="/services"
        style={{
          display: "inline-block",
          padding: "14px 32px",
          background: "#D32F2F",
          color: "#fff",
          fontWeight: 700,
          fontSize: "1rem",
          textTransform: "uppercase",
          letterSpacing: 0.5,
          transition: "background 0.2s",
        }}
      >
        Перейти к точкам наблюдения
      </Link>
    </div>
  );
}
