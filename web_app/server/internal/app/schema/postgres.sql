create table if not exists users (
  id text primary key,
  username text not null unique,
  password_hash text not null,
  created_at timestamptz not null
);

create table if not exists uploads (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  file_name text not null,
  size_bytes bigint not null,
  object_key text not null,
  mesh_kind text not null default 'uploaded',
  point_count integer,
  edge_count integer,
  created_at timestamptz not null
);

create table if not exists jobs (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  upload_id text not null references uploads(id),
  name text,
  input_name text not null,
  input_object_key text not null,
  config jsonb not null,
  status text not null,
  result_object_key text,
  converged boolean not null default false,
  reason text,
  final_time double precision,
  final_step integer,
  error text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  finished_at timestamptz
);

create table if not exists job_snapshots (
  id bigserial primary key,
  job_id text not null references jobs(id) on delete cascade,
  label text not null,
  sim_time double precision not null,
  step integer not null,
  object_key text not null,
  created_at timestamptz not null
);

create table if not exists job_reviews (
  job_id text primary key references jobs(id) on delete cascade,
  user_id text not null references users(id) on delete cascade,
  score integer not null check (score between 1 and 5),
  tags text[] not null default '{}',
  note text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table if not exists training_clusters (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  name text not null,
  status text not null default 'ready',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table if not exists training_cluster_jobs (
  cluster_id text not null references training_clusters(id) on delete cascade,
  job_id text not null references jobs(id) on delete cascade,
  added_at timestamptz not null,
  primary key (cluster_id, job_id)
);

create table if not exists training_runs (
  id text primary key,
  cluster_id text not null references training_clusters(id) on delete cascade,
  status text not null,
  metrics jsonb not null default '{}',
  model_artifact text,
  error text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  finished_at timestamptz
);

create table if not exists config_recommendations (
  id bigserial primary key,
  run_id text not null references training_runs(id) on delete cascade,
  rank integer not null,
  config jsonb not null,
  predicted_score double precision not null,
  predicted_tags jsonb not null default '[]',
  created_at timestamptz not null
);

create unique index if not exists users_username_lower_idx on users (lower(username));
create index if not exists uploads_user_id_created_at_idx on uploads (user_id, created_at desc);
create index if not exists jobs_user_id_created_at_idx on jobs (user_id, created_at desc);
create index if not exists jobs_upload_id_idx on jobs (upload_id);
create index if not exists job_snapshots_job_id_step_idx on job_snapshots (job_id, step);
create index if not exists job_reviews_user_id_updated_at_idx on job_reviews (user_id, updated_at desc);
create index if not exists training_clusters_user_id_created_at_idx on training_clusters (user_id, created_at desc);
create index if not exists training_cluster_jobs_job_id_idx on training_cluster_jobs (job_id);
create index if not exists training_runs_cluster_id_created_at_idx on training_runs (cluster_id, created_at desc);
create index if not exists config_recommendations_run_id_rank_idx on config_recommendations (run_id, rank);
