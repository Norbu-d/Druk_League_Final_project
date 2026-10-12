CREATE TABLE IF NOT EXISTS dzongkhags (
  code text PRIMARY KEY,
  name text NOT NULL UNIQUE
);
INSERT INTO dzongkhags(code,name) VALUES
  ('01','Bumthang'),('02','Chhukha'),('03','Dagana'),('04','Gasa'),('05','Haa'),
  ('06','Lhuntse'),('07','Mongar'),('08','Paro'),('09','Pema Gatshel'),('10','Punakha'),
  ('11','Samdrup Jongkhar'),('12','Samtse'),('13','Sarpang'),('14','Thimphu'),('15','Trashigang'),
  ('16','Trashiyangtse'),('17','Trongsa'),('18','Tsirang'),('19','Wangdue Phodrang'),('20','Zhemgang')
ON CONFLICT (code) DO NOTHING;