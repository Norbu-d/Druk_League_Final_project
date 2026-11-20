ALTER TABLE players ADD COLUMN IF NOT EXISTS search_version bigint NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS search_outbox (
  event_id uuid PRIMARY KEY,
  subject text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
CREATE INDEX IF NOT EXISTS search_outbox_pending_idx ON search_outbox(created_at,event_id) WHERE published_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS search_outbox_entity_version_idx ON search_outbox ((payload->>'entity_id'), ((payload->>'version')::bigint));

UPDATE players SET search_version=1 WHERE search_version=0;
INSERT INTO search_outbox(event_id,subject,payload,created_at)
SELECT event_id,'drukleague.search.player',jsonb_build_object('event_id',event_id,'entity_type','player','entity_id',id,'version',search_version,'operation','upsert','occurred_at',created_at,'fields',jsonb_build_object('title',concat_ws(' ',given_name,family_name),'subtitle',concat_ws(' · ','Team '||team_id::text,NULLIF(shirt_number,'')),'search_text',concat_ws(' ',given_name,family_name,shirt_number,team_id::text))),created_at FROM (SELECT gen_random_uuid() AS event_id, players.* FROM players) snapshot
ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION enqueue_player_search_event() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE event_id uuid := gen_random_uuid(); event_version bigint; event_time timestamptz := clock_timestamp();
BEGIN
  IF TG_OP = 'DELETE' THEN
    event_version := OLD.search_version + 1;
    INSERT INTO search_outbox(event_id,subject,payload,created_at) VALUES(event_id,'drukleague.search.player',jsonb_build_object('event_id',event_id,'entity_type','player','entity_id',OLD.id,'version',event_version,'operation','delete','occurred_at',event_time));
    RETURN OLD;
  END IF;
  IF TG_OP = 'INSERT' THEN NEW.search_version := GREATEST(NEW.search_version,1); ELSE NEW.search_version := OLD.search_version + 1; END IF;
  INSERT INTO search_outbox(event_id,subject,payload,created_at) VALUES(event_id,'drukleague.search.player',jsonb_build_object('event_id',event_id,'entity_type','player','entity_id',NEW.id,'version',NEW.search_version,'operation','upsert','occurred_at',event_time,'fields',jsonb_build_object('title',concat_ws(' ',NEW.given_name,NEW.family_name),'subtitle',concat_ws(' · ','Team '||NEW.team_id::text,NULLIF(NEW.shirt_number,'')),'search_text',concat_ws(' ',NEW.given_name,NEW.family_name,NEW.shirt_number,NEW.team_id::text))));
  RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS players_search_event ON players;
CREATE TRIGGER players_search_event BEFORE INSERT OR UPDATE OR DELETE ON players FOR EACH ROW EXECUTE FUNCTION enqueue_player_search_event();
