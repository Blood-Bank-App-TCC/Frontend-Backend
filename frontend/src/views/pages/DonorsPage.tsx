import { Plus, Search } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodType, Donor } from "../../models/types";
import { bloodTypes } from "../../models/status";
import { DonorTable } from "../components/DataTables";
import PageHeader from "../layout/PageHeader";

export default function DonorsPage() {
  const [donors, setDonors] = useState<Donor[]>([]);
  const [search, setSearch] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [form, setForm] = useState({
    fullName: "",
    bloodType: "O-" as BloodType,
    gender: "M" as Donor["gender"],
    birthDate: "1990-01-01",
    phone: "",
    email: "",
    address: "",
  });

  async function load() {
    setLoading(true);
    try {
      setDonors(await bankDarahController.donors(search));
    } catch (err) {
      setError(apiErrorMessage(err, "Daftar pendonor gagal dimuat."));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, [search]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await bankDarahController.createDonor(form);
      setForm({ fullName: "", bloodType: "O-", gender: "M", birthDate: "1990-01-01", phone: "", email: "", address: "" });
      setShowForm(false);
      setMessage("Pendonor berhasil ditambahkan.");
      await load();
    } catch (err) {
      setError(apiErrorMessage(err, "Pendonor gagal ditambahkan."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <PageHeader
        eyebrow="Master data"
        title="Data Pendonor"
        action={
          <button className="btn primary" type="button" onClick={() => setShowForm((value) => !value)}>
            <Plus size={18} />
            Tambah Donor
          </button>
        }
      />
      <div className="toolbar">
        <Search size={18} />
        <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Cari nama, telepon, atau golongan darah" />
      </div>
      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}
      {showForm && (
        <form className="panel form-grid" onSubmit={submit}>
          <label>
            Nama Lengkap
            <input value={form.fullName} onChange={(event) => setForm({ ...form, fullName: event.target.value })} required />
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
            Gender
            <select value={form.gender} onChange={(event) => setForm({ ...form, gender: event.target.value as Donor["gender"] })}>
              <option value="M">Laki-laki</option>
              <option value="F">Perempuan</option>
            </select>
          </label>
          <label>
            Tanggal Lahir
            <input type="date" value={form.birthDate} onChange={(event) => setForm({ ...form, birthDate: event.target.value })} required />
          </label>
          <label>
            Telepon
            <input value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} required />
          </label>
          <label>
            Email
            <input value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
          </label>
          <label>
            Alamat
            <input value={form.address} onChange={(event) => setForm({ ...form, address: event.target.value })} required />
          </label>
          <div className="form-actions span-2">
            <button className="btn primary" type="submit" disabled={saving}>
              {saving ? "Menyimpan..." : "Simpan Donor"}
            </button>
          </div>
        </form>
      )}
      <section className="panel">
        {loading ? <p className="subtle">Memuat pendonor...</p> : <DonorTable donors={donors} />}
      </section>
    </>
  );
}
