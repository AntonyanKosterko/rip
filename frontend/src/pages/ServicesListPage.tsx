import { useState, useEffect } from "react";
import { mockServices, type ObservationPoint } from "../mock/services";
import ServiceCard from "../components/ServiceCard";
import SearchBar from "../components/SearchBar";

export default function ServicesListPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [filtered, setFiltered] = useState<ObservationPoint[]>(mockServices);

  useEffect(() => {
    if (!searchQuery.trim()) {
      setFiltered(mockServices);
      return;
    }
    const q = searchQuery.trim().toLowerCase();
    setFiltered(
      mockServices.filter(
        (s) =>
          s.name.toLowerCase().includes(q) ||
          s.country.toLowerCase().includes(q)
      )
    );
  }, [searchQuery]);

  return (
    <>
      <SearchBar value={searchQuery} onChange={setSearchQuery} />

      <main className="content">
        <div className="section-header">
          <h2 className="section-title">Точки наблюдения</h2>
          {searchQuery.trim() && (
            <span className="search-result-info">
              Результаты поиска: &laquo;{searchQuery}&raquo; — найдено {filtered.length}
            </span>
          )}
        </div>

        <div className="services-grid">
          {filtered.length > 0 ? (
            filtered.map((svc) => (
              <ServiceCard key={svc.id} service={svc} />
            ))
          ) : (
            <div className="no-results">
              <p>Точки наблюдения не найдены.</p>
              <p>Попробуйте изменить поисковый запрос.</p>
            </div>
          )}
        </div>
      </main>
    </>
  );
}
