import { useState } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import NavBar from "./components/NavBar";
import BreadCrumbs from "./components/BreadCrumbs";
import HomePage from "./pages/HomePage";
import ServicesListPage from "./pages/ServicesListPage";
import ServiceDetailPage from "./pages/ServiceDetailPage";

export default function App() {
  const [cartIds, setCartIds] = useState<number[]>([]);

  const handleAddToCart = (id: number) => {
    setCartIds((prev) => (prev.includes(id) ? prev : [...prev, id]));
  };

  return (
    <BrowserRouter>
      <div className="app-shell">
        <NavBar cartCount={cartIds.length} />
        <BreadCrumbs />
        <div className="app-main">
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/services" element={<ServicesListPage />} />
            <Route
              path="/services/:id"
              element={<ServiceDetailPage onAddToCart={handleAddToCart} />}
            />
          </Routes>
        </div>
        <footer className="main-footer">
          <div className="footer-content">
            <p>&copy; 2026 — Определение времени видимости МКС с Земли</p>
            <p className="footer-sub">Лабораторная работа №5 — РИП, МГТУ им. Н.Э. Баумана</p>
          </div>
        </footer>
      </div>
    </BrowserRouter>
  );
}
