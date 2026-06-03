import { Ambulance } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import { formatIndonesianPhone } from "../../models/phone";
import type { BloodType, Hospital, ProductType, UrgencyLevel } from "../../models/types";
import { bloodTypes, productTypes } from "../../models/status";
import PageHeader from "../layout/PageHeader";

export default function EmergencyNewPage() {
  const navigate = useNavigate();
  const [hospitals, setHospitals] = useState<Hospital[]>([]);
  const [selectedHospitalId, setSelectedHospitalId] = useState("");
  const [form, setForm] = useState({
    hospitalName: "",
    picName: "",
    picPhone: "",
    bloodType: "O-" as BloodType,
    productType: "PRC" as ProductType,
    quantityNeeded: 3,
    urgencyLevel: "CRITICAL" as UrgencyLevel,
    notes: "Perdarahan pascaoperasi, butuh donor secepatnya.",
  });
  const [loading, setLoading] = useState(false);
  const [loadingHospitals, setLoadingHospitals] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    bankDarahController
      .hospitals()
      .then((data) => {
        setHospitals(data);
        if (data.length > 0) {
          applyHospital(data[0]);
        }
      })
      .catch((err) => setError(apiErrorMessage(err, "Daftar rumah sakit gagal dimuat.")))
      .finally(() => setLoadingHospitals(false));
  }, []);

  function applyHospital(hospital: Hospital) {
    setSelectedHospitalId(hospital.id);
    setForm((value) => ({
      ...value,
      hospitalName: hospital.name,
      picName: hospital.picName,
      picPhone: formatIndonesianPhone(hospital.picPhone),
    }));
  }

  function selectHospital(id: string) {
    const hospital = hospitals.find((item) => item.id === id);
    if (hospital) {
      applyHospital(hospital);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const request = await bankDarahController.createRequest(form);
      navigate(`/emergency/${request.id}/broadcast`);
    } catch (err) {
      setError(apiErrorMessage(err, "Permintaan darurat gagal dibuat."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <PageHeader eyebrow="Emergency request" title="Form Permintaan Darurat" />
      {error && <p className="alert danger">{error}</p>}
      <form className="panel form-grid" onSubmit={submit}>
        <label>
          Nama Rumah Sakit
          <select value={selectedHospitalId} onChange={(event) => selectHospital(event.target.value)} required disabled={loadingHospitals}>
            {hospitals.length === 0 && <option value="">{loadingHospitals ? "Memuat rumah sakit..." : "Belum ada rumah sakit"}</option>}
            {hospitals.map((hospital) => (
              <option key={hospital.id} value={hospital.id}>
                {hospital.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Nama PIC
          <input value={form.picName} readOnly required />
        </label>
        <label>
          Nomor Kontak
          <input value={form.picPhone} readOnly required />
        </label>
        <label>
          Golongan Darah
          <select value={form.bloodType} onChange={(event) => setForm({ ...form, bloodType: event.target.value as BloodType })}>
            {bloodTypes.map((type) => (
              <option key={type}>{type}</option>
            ))}
          </select>
        </label>
        <label>
          Produk
          <select value={form.productType} onChange={(event) => setForm({ ...form, productType: event.target.value as ProductType })}>
            {productTypes.map((type) => (
              <option key={type}>{type}</option>
            ))}
          </select>
        </label>
        <label>
          Jumlah Kantong
          <input
            min={1}
            required
            type="number"
            value={form.quantityNeeded}
            onChange={(event) => setForm({ ...form, quantityNeeded: Math.max(1, Number(event.target.value)) })}
          />
        </label>
        <label>
          Tingkat Urgensi
          <select value={form.urgencyLevel} onChange={(event) => setForm({ ...form, urgencyLevel: event.target.value as UrgencyLevel })}>
            <option>CRITICAL</option>
            <option>URGENT</option>
            <option>NORMAL</option>
          </select>
        </label>
        <label className="span-2">
          Catatan
          <textarea value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} />
        </label>
        <div className="form-actions span-2">
          <button className="btn primary" disabled={loading || loadingHospitals || hospitals.length === 0} type="submit">
            <Ambulance size={18} />
            {loading ? "Menyimpan..." : "Cari Donor Eligible"}
          </button>
        </div>
      </form>
    </>
  );
}
