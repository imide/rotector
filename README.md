<h1 align="center">
  <picture>
    <img width="800" alt="Rotector" src="./assets/gif/banner.gif">
  </picture>
  <br>
  <a href="https://github.com/robalyx/rotector/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/robalyx/rotector?style=flat-square&color=4a92e1">
  </a>
  <a href="https://github.com/robalyx/rotector/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/robalyx/rotector/ci.yml?style=flat-square&color=4a92e1">
  </a>
  <a href="https://github.com/robalyx/rotector/issues">
    <img src="https://img.shields.io/github/issues/robalyx/rotector?style=flat-square&color=4a92e1">
  </a>
  <a href="CODE_OF_CONDUCT.md">
    <img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4a92e1?style=flat-square">
  </a>
  <a href="https://discord.gg/2Cn7kXqqhY">
    <img src="https://img.shields.io/discord/1294585467462746292?style=flat-square&color=4a92e1&label=discord" alt="Join our Discord">
  </a>
</h1>

<p align="center">
  <em>When Roblox and Discord moderators dream of superpowers, they dream of <b>Rotector</b>. A powerful system built with <a href="https://go.dev/">Go</a> that uses AI and smart algorithms to find inappropriate Roblox and Discord accounts.</em>
</p>

---

> [!WARNING]
> This project is open-sourced for version control and transparency only. **DO NOT** attempt to set up your own instance as the system requires significant technical expertise, infrastructure resources, and specific configurations to operate effectively. No support or guides are provided.

> [!IMPORTANT]
> This project is currently in an **ALPHA** state with frequent breaking changes - **do not use this in production yet**. This is a **community-driven initiative** and is not affiliated with, endorsed by, or sponsored by Roblox Corporation.

---

## 📚 Table of Contents

- [🛠️ Community Tools](#%EF%B8%8F-community-tools)
- [❓ FAQ](#-faq)
- [📄 License](#-license)

## 🛠️ Community Tools

|                                                                                                                                                                                                        [Rotten - Official Export Checker](https://github.com/robalyx/rotten)                                                                                                                                                                                                        |                                                                                                                                                                                                    [Roscoe - Official REST API](https://github.com/robalyx/roscoe)                                                                                                                                                                                                    |
|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
| <p align="center"><a href="https://github.com/robalyx/rotten"><img src="assets/gif/rotten.gif" height="500"></a></p><p align="center">A simple command-line tool that lets you check Roblox accounts against Rotector exports. Easily verify individual users, groups, or scan entire friend lists for flagged accounts.</p><p align="center">[![GitHub](https://img.shields.io/badge/View_Repository-4a92e1?style=flat-square&logo=github)](https://github.com/robalyx/rotten)</p> | <p align="center"><a href="https://github.com/robalyx/roscoe"><img src="assets/gif/roscoe.gif" height="500"></a></p><p align="center">A globally-distributed REST API powered by Cloudflare Workers and D1. Efficiently stores and serves flagged users from edge locations worldwide for minimal latency.</p><p align="center">[![GitHub](https://img.shields.io/badge/View_Repository-4a92e1?style=flat-square&logo=github)](https://github.com/robalyx/roscoe)</p> |

## ❓ FAQ

<details>
<summary>How can I trust that Rotector's analysis is accurate?</summary>

While Roblox uses AI systems for various moderation tasks like chat moderation, they often struggle with detecting serious issues in user profiles and behavior patterns. For example, their chat moderation system shows high false positive rates without human oversight, which is a problem we've learned from and specifically addressed in our profile analysis approach.

Through months of fine-tuning AI models, detection algorithms, and pattern recognition systems, we've achieved a high level of accuracy in analyzing user profiles and identifying inappropriate accounts.

However, like any automated system, false positives can still occur, particularly with borderline cases where content might be ambiguous or open to interpretation. These cases are typically flagged with lower confidence scores to indicate the need for careful review.

To address this, we have a feature-rich review system where our moderators carefully examine each flagged account through our Discord interface before making final decisions. This human oversight helps ensure that borderline cases are handled appropriately and prevents incorrect flags from affecting future results.

</details>

<details>
<summary>How does the Roblox user detection system work?</summary>

The system operates through multiple sophisticated layers of detection and verification. At its core, specialized workers continuously scan friend lists and groups to identify potentially inappropriate users. These workers analyze various things including membership in inappropriate groups, connections to known inappropriate accounts, avatar outfits, condo game activities, and inappropriate user descriptions. We make use of AI and algorithms to flag accounts and generate reasons.

When users get flagged through these detection methods, they appear in the Discord bot interface where moderators can conduct thorough reviews. The bot provides specialized tools that allow moderators to perform checks before deciding whether to confirm or clear the flag.

The system also includes an API service called [roscoe](https://github.com/robalyx/roscoe) that developers can integrate with their own applications to check user flags. Additionally, we provide a tool called [rotten](https://github.com/robalyx/rotten) that allows anyone to verify user IDs against a hashed list, which even with dedicated effort, would take approximately a year to crack the complete list with standard hardware.

Rotector focuses on identifying and tracking inappropriate accounts, while leaving the actual account moderation and termination decisions to Roblox. Roblox administrators would need to contact us directly to access the uncensored list.

</details>

<details>
<summary>How does the Discord user detection system work?</summary>

Our worker uses undocumented Discord endpoints to scan through all active members of Discord condo servers. We also perform periodic full user scans that allow us to discover every condo server a user is participating in, which helps us maintain an accurate database of user activities .

Server administrators can add the Discord bot into their Discord servers and access guild owner tools through the dashboard. They have two primary moderation options: banning users based on inappropriate server membership or banning users based on inappropriate message content. We strongly recommend the latter.

We specifically advise against banning users solely based on server membership due to the high risk of affecting innocent users. While we do have filters in place like minimum guild counts and minimum join time requirements to improve accuracy, this method can still affect investigators, reporters, non-participating members, compromised accounts, and those who joined through misleading invites. However, we have measurements that only flag users for server membership if they've been present in a condo server for more than 12 hours or have communicated in the server.

Our recommended method which uses an AI system actively analyzes messages sent in these condo servers by examining the full conversation context and user behavior. This approach focuses on actual inappropriate activity rather than making assumptions, providing a much more accurate and fair solution to the moderation challenge compared to systems that rely solely on server membership. This approach differentiates us from systems like Ruben's Ro-Cleaner bot.

</details>

<details>
<summary>How does Rotector handle privacy laws and data protection?</summary>

We take privacy laws and data protection regulations very seriously, setting us apart from other similar initiatives. While we keep historical data of user profiles even after updates for tracking behavior patterns, we have a way to comply with various privacy regulations including GDPR (European Union) and CCPA (California).

Our appeal system serves multiple purposes. Not only can users appeal their flagged account(s), they can also request data deletion, access their stored information, or request an update of their records.

When a data deletion request is received through the appeal system, we carefully validate the request and process it according to the applicable privacy law requirements. However, we may retain certain minimal information if required by law or legitimate interest, such as maintaining records of dangerous offenders.

</details>

<details>
<summary>What AI models does Rotector use?</summary>

Rotector uses the [official OpenAI library](https://github.com/openai/openai-go) for its AI capabilities, which means it can work with any OpenAI-compatible API service like [OpenRouter](https://openrouter.ai/) or [Requesty](https://www.requesty.ai/). While we use Gemini 2.0 Flash by default due to its excellent price-to-performance ratio, you have the flexibility to use any compatible model available through these services.

</details>

<details>
<summary>Why was Go chosen over Python or JavaScript?</summary>

A small prototype of Rotector was actually built in JavaScript several months before the project officially started. While it worked for testing basic concepts, we quickly hit limitations when trying to process thousands of accounts efficiently as it would slow down significantly due to its single threaded nature.

Go was the perfect solution to these challenges. It's incredibly good at running thousands of concurrent tasks (called goroutines) that are far more lightweight and efficient than Python's threads or JavaScript's async/await. These goroutines allow us to process massive amounts of data while using significantly less memory and CPU resources.

Go's built-in type system also provides compile-time safety without the overhead of TypeScript's runtime checks. While TypeScript offers similar features, Go's native implementation means faster execution and better performance. The compiler catches potential issues before deployment, making the system more reliable in production.

Another huge advantage is Go's code generation capabilities. Instead of writing hundreds of lines of repetitive code by hand for things like our session management system, Go generates it automatically, making the code cleaner and more reliable.

Go's superior memory management with minimal garbage collection pauses, fast compilation times, and single binary deployments also make it ideal for our large-scale system. Its standard library provides everything we need without relying heavily on external dependencies.

</details>

<details>
<summary>Why use Discord instead of a custom web interface?</summary>

Discord already has everything we need for reviewing accounts like buttons, dropdowns, forms, and rich embeds. Using Discord lets us focus on making Rotector better instead of building a whole new website interface from scratch.

</details>

<details>
<summary>Will users who have stopped their inappropriate behavior be removed from the database?</summary>

No, confirmed and flagged users remain in the database permanently, even if they're banned or claim to have changed. This data retention helps track patterns of behavior and can be valuable for law enforcement investigations or identifying repeat offenders.

We have a system in place to manage privacy laws. While users can request their data to be deleted through our appeal system in accordance with privacy regulations like GDPR and CCPA, their flag status will remain in our database regardless.

</details>

<details>
<summary>What's the story behind Rotector?</summary>

While many community initiatives have made valiant efforts to address Roblox's safety concerns, they've often been limited by technical expertise and resource constraints. Despite initially believing it was impossible for a single person to tackle such a massive platform-wide issue affecting millions of users, [jaxron](https://github.com/jaxron) took the first step by developing Rotector's initial prototype on October 13, 2024, driven by growing concerns about Roblox's moderation challenges and a desire to protect young players.

The foundation for this project was laid a few weeks earlier when jaxron developed two crucial libraries: [RoAPI.go](https://github.com/jaxron/roapi.go) and [axonet](https://github.com/jaxron/axonet) on September 23, 2024, which would become Rotector's core networking capabilities. The project went public for alpha testing on November 8, 2024.

While Roblox already has moderators, the scale of the platform makes it hard to catch every inappropriate account. Even Roblox staff have acknowledged the difficulties in handling the reports they receive, sometimes leading to inappropriate content staying active even after reports.

Rotector aims to bridge this gap by automating the detection process, making moderation more efficient and helping create a safer environment for the Roblox community. The story is still being written, and we're excited about the upcoming release.

</details>

## 📄 License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  🚀 <strong>Protection at scale, when others fail.</strong>
</p>
