import Hero from './components/Hero/Hero'
import Instructions from './components/Instructions/Instructions'
import styles from './App.module.css'

function App() {
  return (
    <main className={styles.page}>
      <Hero />
      <Instructions />
    </main>
  )
}

export default App
