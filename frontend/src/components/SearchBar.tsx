interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
}

export default function SearchBar({ value, onChange }: SearchBarProps) {
  return (
    <div className="sub-header">
      <div className="sub-header-content">
        <form
          className="search-form"
          onSubmit={(e) => e.preventDefault()}
        >
          <input
            type="text"
            className="search-input"
            placeholder="Поиск по названию точки..."
            value={value}
            onChange={(e) => onChange(e.target.value)}
          />
          <button type="submit" className="search-btn">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </button>
        </form>
      </div>
    </div>
  );
}
