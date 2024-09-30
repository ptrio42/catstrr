import express from 'express';
import { pipeline, env } from '@xenova/transformers';

// env.allowLocalModels = true;
// env.allowRemoteModels = false;
// env.userBrowserCache = false;

class Pipeline {
  static task = 'object-detection';
  static model = 'Xenova/detr-resnet-50';
  static instance = null;

  static async getInstance(progressCallback = null) {
    if (this.instance === null) {
      this.instance = await pipeline(this.task, this.model, { progress_callback: progressCallback });
    }
    return this.instance;
  }
}

const app = express();

app.use(express.json());

app.post('/process-image', async (req, res) => {
  const image = req.body.image;
  env.cacheDir = './.cache';
  const detector = await Pipeline.getInstance();
  const response = await detector(image);
  const isCatImage = response.findIndex(({label, score}) => label === 'cat') > -1;
  console.log({response})
  res.json({ result: isCatImage });
});

const PORT = 3003;
app.listen(PORT, () => {
  console.log(`Server is running on http://localhost:${PORT}`);
});

