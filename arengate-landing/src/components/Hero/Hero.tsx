import styles from './Hero.module.css'

export default function Hero() {
  return (
    <section className={styles.hero} aria-label="Arendelle Gate Tech">
      <div className={styles.overlay} />
      <div className={styles.content}>
        <h1 className={styles.title}>Arendelle Gate Tech</h1>
        <p className={styles.subtitle}>Your gateway to free internet</p>
      </div>
      <div className={styles.scrollHint} aria-hidden="true">
        ↓
      </div>
    </section>
  )
}
