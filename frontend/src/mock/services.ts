export interface ObservationPoint {
  id: number;
  name: string;
  country: string;
  latitude: number;
  longitude: number;
  elevation: number;
  timezone: string;
  best_time: string;
  light_pollution: string;
  weather_conditions: string;
  description: string;
  image_url: string;
  video_url: string;
}

const DEFAULT_IMAGE = "/media/moscow.jpg";

export const mockServices: ObservationPoint[] = [
  {
    id: 1,
    name: "Москва",
    country: "Россия",
    latitude: 55.7558,
    longitude: 37.6173,
    elevation: 156,
    timezone: "UTC+3 (МСК)",
    best_time: "Март — апрель, сентябрь — октябрь",
    light_pollution: "Высокая",
    weather_conditions: "Переменная облачность, лучшие условия в ясные зимние ночи",
    description:
      "Москва — столица Российской Федерации и один из крупнейших мегаполисов мира. Несмотря на высокий уровень световой загрязнённости, МКС можно наблюдать невооружённым глазом благодаря её яркости. Рекомендуется выбирать наблюдательные площадки на возвышенностях — Воробьёвы горы, смотровая площадка МГУ или парк «Коломенское».",
    image_url: "/media/moscow.jpg",
    video_url: "/media/moscow_video.mp4",
  },
  {
    id: 2,
    name: "Байконур",
    country: "Казахстан",
    latitude: 45.9647,
    longitude: 63.305,
    elevation: 90,
    timezone: "UTC+6",
    best_time: "Круглый год, лучше всего летом",
    light_pollution: "Низкая",
    weather_conditions: "Преимущественно ясная погода, сухой континентальный климат",
    description:
      "Космодром Байконур — первый и крупнейший в мире космодром. Именно отсюда 12 апреля 1961 года стартовал корабль «Восток-1» с Юрием Гагариным на борту. Благодаря удалённости от крупных городов и сухому климату степной зоны, здесь идеальные условия для наблюдения за МКС.",
    image_url: "/media/baikonur.jpg",
    video_url: "/media/baikonur_video.mp4",
  },
  {
    id: 3,
    name: "Мыс Канаверал",
    country: "США",
    latitude: 28.3922,
    longitude: -80.6077,
    elevation: 3,
    timezone: "UTC−5 (EST)",
    best_time: "Ноябрь — февраль",
    light_pollution: "Средняя",
    weather_conditions: "Субтропический климат, частая облачность летом",
    description:
      "Мыс Канаверал — легендарная стартовая площадка NASA на атлантическом побережье штата Флорида. Здесь расположены Космический центр Кеннеди и база ВВС США, откуда осуществлялись запуски программ «Аполлон», Space Shuttle и современных ракет SpaceX.",
    image_url: "/media/cape_canaveral.jpg",
    video_url: "/media/cape_canaveral_video.mp4",
  },
  {
    id: 4,
    name: "Куру",
    country: "Французская Гвиана",
    latitude: 5.1594,
    longitude: -52.6503,
    elevation: 8,
    timezone: "UTC−3",
    best_time: "Август — ноябрь (сухой сезон)",
    light_pollution: "Низкая",
    weather_conditions: "Экваториальный климат, высокая влажность, частые дожди",
    description:
      "Гвианский космический центр в городе Куру — основной космодром Европейского космического агентства (ESA). Расположен всего в 5 градусах от экватора.",
    image_url: "/media/kourou.jpg",
    video_url: "/media/kourou_video.mp4",
  },
  {
    id: 5,
    name: "Восточный",
    country: "Россия",
    latitude: 51.8844,
    longitude: 128.334,
    elevation: 275,
    timezone: "UTC+9",
    best_time: "Июнь — август",
    light_pollution: "Очень низкая",
    weather_conditions: "Континентальный климат, холодные зимы, тёплое лето",
    description:
      "Космодром Восточный — новейший российский космодром, расположенный в Амурской области Дальнего Востока. Строительство началось в 2012 году, первый пуск состоялся 28 апреля 2016 года.",
    image_url: "/media/vostochny.jpg",
    video_url: "/media/vostochny_video.mp4",
  },
  {
    id: 6,
    name: "Танегасима",
    country: "Япония",
    latitude: 30.399,
    longitude: 130.9707,
    elevation: 20,
    timezone: "UTC+9 (JST)",
    best_time: "Декабрь — март",
    light_pollution: "Низкая",
    weather_conditions: "Субтропический океанический климат, мягкие зимы",
    description:
      "Космический центр Танегасима (TNSC) — крупнейший космодром Японии, управляемый агентством JAXA. Расположен на южной оконечности острова Танегасима в префектуре Кагосима.",
    image_url: "/media/tanegashima.jpg",
    video_url: "/media/tanegashima_video.mp4",
  },
];

export { DEFAULT_IMAGE };

export function coordinates(p: ObservationPoint): string {
  return `${p.latitude.toPrecision(4)}°, ${p.longitude.toPrecision(4)}°`;
}
