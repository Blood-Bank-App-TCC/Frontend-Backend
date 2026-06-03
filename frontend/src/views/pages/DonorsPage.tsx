import { Search } from "lucide-react";
import { useEffect, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { Donor } from "../../models/types";
import { DonorTable } from "../components/DataTables";
import PageHeader from "../layout/PageHeader";

export default function DonorsPage() {
  const [donors, setDonors] = useState<Donor[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

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

  return (
    <>
      <PageHeader eyebrow="Master data" title="Data Pendonor" />
      <div className="toolbar">
        <Search size={18} />
        <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Cari nama, telepon, atau golongan darah" />
      </div>
      {error && <p className="alert danger">{error}</p>}
      <section className="panel">
        {loading ? <p className="subtle">Memuat pendonor...</p> : <DonorTable donors={donors} />}
      </section>
    </>
  );
}
