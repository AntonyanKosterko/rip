import { Link } from "react-router-dom";
import { coordinates, DEFAULT_IMAGE } from "../mock/services";
import type { ObservationPoint } from "../mock/services";

interface ServiceCardProps {
  service: ObservationPoint;
}

export default function ServiceCard({ service }: ServiceCardProps) {
  const imgSrc = service.image_url || DEFAULT_IMAGE;

  return (
    <Link to={`/services/${service.id}`} className="service-card" data-service-id={service.id}>
      <div className="card-image">
        <img
          src={imgSrc}
          alt={service.name}
          loading="lazy"
          onError={(e) => { (e.target as HTMLImageElement).src = DEFAULT_IMAGE; }}
        />
        <div className="card-image-overlay" />
        <span className="card-label">Наблюдение</span>
        <span className="card-coords-badge">{coordinates(service)}</span>
      </div>
      <div className="card-content">
        <h3 className="card-title">{service.name}</h3>
        <p className="card-country">{service.country}</p>
        <div className="card-meta">
          <span className="card-elevation">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14 6l-3.75 5 2.85 3.8-1.6 1.2C9.81 13.75 7 10 7 10l-6 8h22z" />
            </svg>
            {service.elevation} м
          </span>
          <span className="card-timezone">{service.timezone}</span>
        </div>
      </div>
    </Link>
  );
}
