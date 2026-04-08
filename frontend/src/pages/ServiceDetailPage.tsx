import { useParams, Link } from "react-router-dom";
import { useState, useEffect } from "react";
import { mockServices, coordinates, DEFAULT_IMAGE } from "../mock/services";
import type { ObservationPoint } from "../mock/services";

interface ServiceDetailPageProps {
  onAddToCart: (id: number) => void;
}

export default function ServiceDetailPage({ onAddToCart }: ServiceDetailPageProps) {
  const { id } = useParams<{ id: string }>();
  const [service, setService] = useState<ObservationPoint | null>(null);
  const [videoFailed, setVideoFailed] = useState(false);

  useEffect(() => {
    const found = mockServices.find((s) => s.id === Number(id)) ?? null;
    setService(found);
    setVideoFailed(false);
  }, [id]);

  if (!service) {
    return (
      <main className="content" style={{ textAlign: "center", padding: "48px 24px" }}>
        <p style={{ color: "#607D8B", fontSize: "1.1rem", marginBottom: 16 }}>Точка наблюдения не найдена.</p>
        <Link to="/services" className="link-accent">← Вернуться к списку</Link>
      </main>
    );
  }

  const imgSrc = service.image_url || DEFAULT_IMAGE;
  const showVideo = Boolean(service.video_url) && !videoFailed;

  return (
    <main className="content">
      <div className="detail-page vibes-layout">

        {/* Vibes-карточка */}
        <div className="vibes-card">
          {showVideo ? (
            <video
              autoPlay
              muted
              loop
              playsInline
              className="vibes-media"
              onError={() => setVideoFailed(true)}
            >
              <source src={service.video_url} type="video/mp4" />
            </video>
          ) : (
            <img
              src={imgSrc}
              alt={service.name}
              className="vibes-media"
              onError={(e) => { (e.target as HTMLImageElement).src = DEFAULT_IMAGE; }}
            />
          )}
          <div className="vibes-overlay" />
          <div className="vibes-content">
            <span className="vibes-label">Точка наблюдения</span>
            <h1 className="vibes-title">{service.name}</h1>
            <p className="vibes-subtitle">{service.country}</p>
            <div className="vibes-coords">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
              </svg>
              <span>Координаты: {coordinates(service)}</span>
            </div>
          </div>

          {/* Боковые иконки */}
          <div className="vibes-side-actions">
            <div className="vibes-action">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M14 6l-3.75 5 2.85 3.8-1.6 1.2C9.81 13.75 7 10 7 10l-6 8h22z" />
              </svg>
              <span>{service.elevation}м</span>
            </div>
            <div className="vibes-action">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              <span>{service.timezone}</span>
            </div>
            <div className="vibes-action">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="5" />
                <line x1="12" y1="1" x2="12" y2="3" />
                <line x1="12" y1="21" x2="12" y2="23" />
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                <line x1="1" y1="12" x2="3" y2="12" />
                <line x1="21" y1="12" x2="23" y2="12" />
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
              </svg>
              <span>{service.light_pollution}</span>
            </div>
            <button
              type="button"
              className="vibes-action vibes-add-btn"
              title="Добавить в заявку"
              onClick={() => onAddToCart(service.id)}
            >
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="3" />
                <line x1="3" y1="12" x2="9" y2="12" />
                <line x1="15" y1="12" x2="21" y2="12" />
                <rect x="1" y="10" width="4" height="4" rx="0.5" />
                <rect x="19" y="10" width="4" height="4" rx="0.5" />
                <line x1="12" y1="9" x2="12" y2="5" />
                <circle cx="12" cy="4" r="1" />
              </svg>
              <span>В заявку</span>
            </button>
          </div>
        </div>

        {/* Характеристики */}
        <div className="detail-info-section">
          <h2 className="detail-section-title">Характеристики</h2>
          <div className="info-grid">
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
                </svg>
              </div>
              <span className="info-label">Координаты</span>
              <span className="info-value">{coordinates(service)}</span>
            </div>
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M14 6l-3.75 5 2.85 3.8-1.6 1.2C9.81 13.75 7 10 7 10l-6 8h22z" />
                </svg>
              </div>
              <span className="info-label">Высота</span>
              <span className="info-value">{service.elevation} м</span>
            </div>
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z" />
                </svg>
              </div>
              <span className="info-label">Часовой пояс</span>
              <span className="info-value">{service.timezone}</span>
            </div>
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6.76 4.84l-1.8-1.79-1.41 1.41 1.79 1.79zM4 10.5H1v2h3zm9-9.95h-2V3.5h2zm7.45 3.91l-1.41-1.41-1.79 1.79 1.41 1.41zm-3.21 13.7l1.79 1.8 1.41-1.41-1.8-1.79zM20 10.5v2h3v-2zm-8-5c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6zm-1 16.95h2V19.5h-2zm-7.45-3.91l1.41 1.41 1.79-1.8-1.41-1.41z" />
                </svg>
              </div>
              <span className="info-label">Лучшее время</span>
              <span className="info-value">{service.best_time}</span>
            </div>
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 3c-4.97 0-9 4.03-9 9s4.03 9 9 9c.83 0 1.5-.67 1.5-1.5 0-.39-.15-.74-.39-1.01-.23-.26-.38-.61-.38-1 0-.83.67-1.5 1.5-1.5H16c2.76 0 5-2.24 5-5 0-4.42-4.03-8-9-8zm-5.5 9c-.83 0-1.5-.67-1.5-1.5S5.67 9 6.5 9 8 9.67 8 10.5 7.33 12 6.5 12zm3-4C8.67 8 8 7.33 8 6.5S8.67 5 9.5 5s1.5.67 1.5 1.5S10.33 8 9.5 8zm5 0c-.83 0-1.5-.67-1.5-1.5S13.67 5 14.5 5s1.5.67 1.5 1.5S15.33 8 14.5 8zm3 4c-.83 0-1.5-.67-1.5-1.5S16.67 9 17.5 9s1.5.67 1.5 1.5-.67 1.5-1.5 1.5z" />
                </svg>
              </div>
              <span className="info-label">Засветка</span>
              <span className="info-value">{service.light_pollution}</span>
            </div>
            <div className="info-card">
              <div className="info-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z" />
                </svg>
              </div>
              <span className="info-label">Погода</span>
              <span className="info-value">{service.weather_conditions}</span>
            </div>
          </div>
        </div>

        {/* Описание */}
        {service.description && (
          <div className="detail-description-section">
            <h2 className="detail-section-title">Описание</h2>
            <p className="detail-description-text">{service.description}</p>
          </div>
        )}
      </div>
    </main>
  );
}
