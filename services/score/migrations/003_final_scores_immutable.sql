CREATE OR REPLACE FUNCTION reject_finalized_score_mutation() RETURNS trigger AS $$
BEGIN
  IF OLD.finalized_at IS NOT NULL THEN
    RAISE EXCEPTION 'finalized score is immutable' USING ERRCODE='23514';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'scores_finalized_immutable'
      AND tgrelid = 'scores'::regclass
      AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER scores_finalized_immutable
      BEFORE UPDATE OR DELETE ON scores
      FOR EACH ROW EXECUTE FUNCTION reject_finalized_score_mutation();
  END IF;
END;
$$;
