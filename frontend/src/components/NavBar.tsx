import { Link } from "react-router-dom";

interface NavBarProps {
  cartCount: number;
}

export default function NavBar({ cartCount }: NavBarProps) {
  return (
    <header className="main-header">
      <div className="header-content">
        <Link to="/" className="home-btn" title="На главную — Видимость МКС">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
            <polyline points="9 22 9 12 15 12 15 22" />
          </svg>
          <span className="home-btn-text">Видимость МКС</span>
        </Link>

        {cartCount > 0 ? (
          <Link to="/cart" className="cart-icon" title="Текущая заявка">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="3" />
              <line x1="3" y1="12" x2="9" y2="12" />
              <line x1="15" y1="12" x2="21" y2="12" />
              <rect x="1" y="10" width="4" height="4" rx="0.5" />
              <rect x="19" y="10" width="4" height="4" rx="0.5" />
              <line x1="12" y1="9" x2="12" y2="5" />
              <circle cx="12" cy="4" r="1" />
            </svg>
            <span className="cart-badge">{cartCount}</span>
          </Link>
        ) : (
          <span className="cart-icon cart-inactive" title="Нет активной заявки">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" opacity="0.4">
              <circle cx="12" cy="12" r="3" />
              <line x1="3" y1="12" x2="9" y2="12" />
              <line x1="15" y1="12" x2="21" y2="12" />
              <rect x="1" y="10" width="4" height="4" rx="0.5" />
              <rect x="19" y="10" width="4" height="4" rx="0.5" />
              <line x1="12" y1="9" x2="12" y2="5" />
              <circle cx="12" cy="4" r="1" />
            </svg>
            <span className="cart-badge cart-badge-empty">0</span>
          </span>
        )}
      </div>
    </header>
  );
}
