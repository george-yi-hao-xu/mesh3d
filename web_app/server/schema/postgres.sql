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

create index if not exists users_username_lower_idx on users (lower(username));
create index if not exists uploads_user_id_created_at_idx on uploads (user_id, created_at desc);
create index if not exists jobs_user_id_created_at_idx on jobs (user_id, created_at desc);
create index if not exists jobs_upload_id_idx on jobs (upload_id);
create index if not exists job_snapshots_job_id_step_idx on job_snapshots (job_id, step);
