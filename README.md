# GoBDS – A Minecraft Bedrock BDS Enhancer ⚡

**GoBDS** aims to supercharge your Minecraft Bedrock Dedicated Server (BDS) by providing enhanced functionality such as IP filtering, whitelisting, custom slash commands, packet filtering, rate limiting, and more! 

This project was originally designed for use in **PokeBedrock** servers, which serve hundreds of thousands of players, but it can be easily adapted to any Bedrock server environment. Join us and experience a new level of server control!

## ✨ Features

Inspired by the original [vanilla-proxy](https://github.com/smell-of-curry/vanilla-proxy), **GoBDS** builds upon those features and adds more exciting capabilities:

- **Command Handling** ⚙️  
  Reads input text packets containing commands (passed by the Scripting API on the downstream server) and processes them efficiently.  
  → *Check out* [Commands.md](./docs/Commands.md)

- **Client Blocking & Claims** ❌  
  Prevents invalid or unwanted breaking/building actions from ever reaching the downstream server, smoothing out performance.  
  → *Learn more in* [Claims.md](./docs/Claims.md)

- **Auto Entity Name Translations** 🌐  
  Adds the possibility of dynamic name tags for entities, bypassing current Scripting API limitations.  
  → *Details in* [NameTags.md](./docs/NameTags.md)

- **Duplication Protection** 🛡️  
  Stops known duplication glitches at the packet level before they can cause havoc.

- **XUID Mapping** 🔗  
  Ensures that even when proxying, XUIDs are properly tracked and displayed on the downstream server.

- **Sign Edit Logging** 🪧  
  Logs all sign edits (a feature BDS doesn’t support natively) for better moderation and server auditing.

- **Custom Borders** 🌍  
  Prevents players from loading or generating chunks outside specified borders. Shows a clean visual border to users.

- **Auto Resource Pack Downloading** 📦  
  Offers seamless resource pack downloading so your players have the content they need, when they need it.

- **Staff Slots & Queue System** ⏳  
  Manages player queues and ensures staff members always have room to join.

- **Ping Indicator** ⏳  
  Sends a players ping every 20 ticks as a title packet to be displayed with a resource pack.  
  → *See* [PingIndicator.md](./docs/PingIndicator.md)

## 🔧 Design Philosophy

**GoBDS** is designed to be **highly configurable** and easily extensible. Every feature includes comprehensive documentation and can be adapted to fit any server’s architecture. If you’re looking to build or scale up your server framework, **GoBDS** is a robust and flexible choice.

## 🚀 Getting Started

TODO

## 🤝 Contributing

We welcome contributions from the community! Feel free to open an issue or submit a pull request if you have suggestions, improvements, or bug fixes.

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/AmazingFeature`).
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`).
4. Push to the branch (`git push origin feature/AmazingFeature`).
5. Open a pull request.

## ❤️ Thanks & Credits

A huge thank you to the [vanilla-proxy](https://github.com/smell-of-curry/vanilla-proxy) project for laying the groundwork. Additional thanks to the **PokeBedrock** community for testing and using **GoBDS** at scale.