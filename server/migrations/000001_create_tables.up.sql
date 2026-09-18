-- comic_events: synced from allcpp.cn
CREATE TABLE IF NOT EXISTS comic_events (
  id BIGSERIAL PRIMARY KEY,
  allcpp_id INTEGER UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  location VARCHAR(100),
  venue TEXT,
  start_date TIMESTAMPTZ NOT NULL,
  end_date TIMESTAMPTZ NOT NULL,
  cover_url TEXT,
  tags TEXT[] DEFAULT '{}',
  type_name VARCHAR(50),
  status VARCHAR(20) NOT NULL DEFAULT 'upcoming',
  raw_data JSONB,
  synced_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_events_status_date ON comic_events(status, start_date);

-- banners: manually managed carousel
CREATE TABLE IF NOT EXISTS banners (
  id BIGSERIAL PRIMARY KEY,
  image_url TEXT NOT NULL,
  title VARCHAR(255),
  link_type VARCHAR(20) DEFAULT 'event',
  link_id INTEGER,
  sort_order INTEGER DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- photographers
CREATE TABLE IF NOT EXISTS photographers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  avatar TEXT,
  description TEXT,
  location VARCHAR(100),
  rating DECIMAL(2,1) DEFAULT 0.0,
  review_count INTEGER DEFAULT 0,
  order_count INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_photographers_rating ON photographers(rating DESC);

-- works
CREATE TABLE IF NOT EXISTS works (
  id BIGSERIAL PRIMARY KEY,
  photographer_id INTEGER NOT NULL REFERENCES photographers(id),
  title VARCHAR(255) NOT NULL,
  images TEXT[] DEFAULT '{}',
  description TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_works_photographer ON works(photographer_id);

-- tags
CREATE TABLE IF NOT EXISTS tags (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  usage_count INTEGER DEFAULT 0
);

-- photographer_tags junction
CREATE TABLE IF NOT EXISTS photographer_tags (
  photographer_id INTEGER NOT NULL REFERENCES photographers(id),
  tag_id INTEGER NOT NULL REFERENCES tags(id),
  PRIMARY KEY (photographer_id, tag_id)
);

-- services (global catalog, not per-photographer for now)
CREATE TABLE IF NOT EXISTS services (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  price INTEGER NOT NULL,
  description TEXT,
  duration INTEGER NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- reviews
CREATE TABLE IF NOT EXISTS reviews (
  id BIGSERIAL PRIMARY KEY,
  photographer_id INTEGER NOT NULL REFERENCES photographers(id),
  user_id INTEGER NOT NULL,
  user_name VARCHAR(100),
  user_avatar TEXT,
  rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
  content TEXT,
  images TEXT[] DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reviews_photographer ON reviews(photographer_id);

-- bookings (schema only, no endpoints in this phase)
CREATE TABLE IF NOT EXISTS bookings (
  id BIGSERIAL PRIMARY KEY,
  photographer_id INTEGER NOT NULL REFERENCES photographers(id),
  coser_id INTEGER NOT NULL,
  service_id INTEGER NOT NULL REFERENCES services(id),
  date DATE NOT NULL,
  time VARCHAR(10) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  total_price INTEGER NOT NULL,
  remarks TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
