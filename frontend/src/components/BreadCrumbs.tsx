import { Link, useLocation } from "react-router-dom";
import { mockServices } from "../mock/services";

interface Crumb {
  label: string;
  path: string;
}

function buildCrumbs(pathname: string): Crumb[] {
  const crumbs: Crumb[] = [{ label: "Главная", path: "/" }];
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length === 0) return crumbs;

  if (segments[0] === "services") {
    crumbs.push({ label: "Точки наблюдения", path: "/services" });
    if (segments.length >= 2) {
      const id = Number(segments[1]);
      const svc = mockServices.find((s) => s.id === id);
      crumbs.push({
        label: svc ? svc.name : `#${segments[1]}`,
        path: `/services/${segments[1]}`,
      });
    }
  }

  return crumbs;
}

export default function BreadCrumbs() {
  const { pathname } = useLocation();
  const crumbs = buildCrumbs(pathname);

  if (crumbs.length <= 1) return null;

  return (
    <nav className="breadcrumbs-nav" style={{
      maxWidth: 1280, margin: "0 auto", padding: "12px 24px 0",
      display: "flex", gap: 8, fontSize: "0.85rem",
    }}>
      {crumbs.map((c, i) => {
        const isLast = i === crumbs.length - 1;
        return (
          <span key={c.path}>
            {i > 0 && <span style={{ color: "#607D8B", margin: "0 4px" }}>/</span>}
            {isLast ? (
              <span style={{ color: "#B0BEC5" }}>{c.label}</span>
            ) : (
              <Link to={c.path} className="link-accent">{c.label}</Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}
